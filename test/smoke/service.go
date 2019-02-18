package smoke

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Service represents a binary that is intended to be run as a long-running
// process. It includes things like crash detection.
type Service struct {
	Bin
	Cmd             *exec.Cmd
	ReadyForCleanup chan struct{}
	Stopped         chan struct{}
	waitOnce        sync.Once
	waitChanMu      sync.Mutex
	waitResult      waitResult
	doneWaiting     chan struct{}
}

// NewService returns a new service bound to the provided binary.
func NewService(bin Bin) *Service {
	bin.ShouldStillBeRunningAfterTest = true
	return &Service{
		Bin:             bin,
		ReadyForCleanup: make(chan struct{}),
		Stopped:         make(chan struct{}),
		doneWaiting:     make(chan struct{}),
	}
}

// Start starts this service.
func (s *Service) Start(t *testing.T, subcmd string, flags Flags, args ...string) {
	t.Helper()
	prepared, err := s.Command(subcmd, flags, args...)
	if err != nil {
		t.Fatal(err)
	}

	prepared.PreRun()

	s.Cmd = prepared.Cmd
	if err := s.Cmd.Start(); err != nil {
		t.Fatalf("error starting %q: %s", s.InstanceName, err)
	}

	if s.Cmd.Process == nil {
		t.Fatalf("[ERROR:%s] cmd.Process nil after cmd.Start", s.ID())
	}
	writePID(t, s.Cmd.Process.Pid)

	go s.wait(t, 10, false)
}

func (s *Service) wait(t *testing.T, count int, looping bool) bool {
	defer close(s.ReadyForCleanup)
	s.debug("waiting for test to finish or process to exit early")
	select {
	// In this case the process ended before the test finished.
	case wr := <-s.waitChan():
		s.debug("got wait result %s", wr)
		exitCode, err := wr.exitCode()
		if err != nil {
			log.Panicf("[PANIC:%s]: unable to determine exit code: %s", s.ID(), err)
		}
		s.debug("exited with code %d; error: %v; logs follow:", exitCode, err)
		s.DumpTail(t, 3)
		s.debug("end logs")
		safeClose(s.Stopped)
		return true
	case <-s.Finished.Failed:
		s.debug("process still running when test failed")
		return false
	case <-s.Finished.Passed:
		s.debug("process still runnung when test passed")
		return false
	}
}

type waitResult struct {
	err error
	ps  *os.ProcessState
}

func fmtWaitStatus(ws syscall.WaitStatus) string {
	return fmt.Sprintf("exited: %t; signal: %s; stopped: %t; coredump: %t; signalled: %t; continued: %t; trapCause: %d; exitstatus: %d; stopsignal: %s",
		ws.Exited(),
		ws.Signal(),
		ws.Stopped(),
		ws.CoreDump(),
		ws.Signaled(),
		ws.Continued(),
		ws.TrapCause(),
		ws.ExitStatus(),
		ws.StopSignal(),
	)
}

func (wr *waitResult) String() string {
	return fmt.Sprintf("error: %v; WaitStatus: %s", wr.err, fmtWaitStatus(wr.ps.Sys().(syscall.WaitStatus)))
}

func (wr *waitResult) exitCode() (int, error) {
	if wr.err == nil {
		return 0, nil
	}
	ee, ok := wr.err.(*exec.ExitError)
	if !ok {
		return -2, fmt.Errorf("wait failed, returned a %T: %s", wr.err, wr.err)
	}
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok {
		return -3, fmt.Errorf("expected *exec.ExitError.Sys() to return a syscall.WaitStatus but got a %T", ee.Sys())
	}

	if ws.ExitStatus() > 0 {
		return ws.ExitStatus(), nil
	}

	return ws.ExitStatus(), fmt.Errorf("invalid failure exit status %d", ws.ExitStatus())
}

func (wr *waitResult) isCrash() bool {
	err, ps := wr.err, wr.ps
	if err != nil {
		return true
	}
	return err != nil || (ps.Exited() && !ps.Success())
}

// waitChan returns a channel that sends one copy of the result of waiting on
// this process, and then is closed. It is safe to call waitChan concurrently
// and only one wait will actually be done on the process.
func (s *Service) waitChan() <-chan waitResult {
	s.waitChanMu.Lock()
	defer s.waitChanMu.Unlock()
	s.waitOnce.Do(func() {
		go func() {
			var wr waitResult
			wr.err = s.Cmd.Wait()
			if !s.Cmd.ProcessState.Exited() {
				log.Panicf("[PANIC:%s] process not exited after wait", s.ID())
			}
			s.waitResult = wr
			close(s.doneWaiting)
		}()
	})
	<-s.doneWaiting
	c := make(chan waitResult, 1)
	defer close(c)
	c <- s.waitResult
	return c
}

func tryGetExitCode(fromDesc string, from interface{}) int {
	if ws, ok := from.(syscall.WaitStatus); ok {
		rtLog("GOT EXIT CODE: %s is a syscall.WaitStatus: %d", fromDesc, ws.ExitStatus())
		return ws.ExitStatus()
	}
	rtLog("UNABLE TO GET EXIT CODE: %s was a %T", fromDesc, from)
	return -3
}

func safeClose(c chan struct{}) {
	select {
	default:
		close(c)
	case <-c:
		// Already closed.
	}
}

func closed(c chan struct{}) bool {
	select {
	default:
		return false
	case <-c:
		return true
	}
}

func (s *Service) debug(f string, a ...interface{}) {
	pid := "not-started"
	if s.Cmd != nil && s.Cmd.Process != nil {
		pid = strconv.Itoa(s.Cmd.Process.Pid)
	}
	rtLog("[DEBUG:"+s.ID()+";pid:"+pid+"] "+f, a...)
}

// Stop stops this service.
func (s *Service) Stop() error {
	if closed(s.Stopped) {
		s.debug("not stopping as already stopped")
		return nil
	}
	defer safeClose(s.Stopped)
	if s.Cmd == nil || s.Cmd.Process == nil {
		s.debug("process is nil; cannot stop")
		return fmt.Errorf("cannot stop %s (not started)", s.InstanceName)
	}
	s.debug("got process")
	waitErr := make(chan error, 1)
	go func() {
		defer close(waitErr)
		var err error
		s.debug("waiting for exit")
		wr := <-s.waitChan()
		err = wr.err
		if err != nil {
			s.debug("ERROR: wait failed: %s", err)
			waitErr <- fmt.Errorf("wait failed: %s", err)
		}
	}()
	// TODO: make timeout configurable
	timeout := time.Second
	s.debug("sending SIGTERM in %s", timeout)
	time.Sleep(timeout)
	s.debug("sending SIGTERM now!")
	if err := s.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		s.debug("error sending SIGTERM: %s", err)
		return fmt.Errorf("sending SIGTERM: %s", err)
	}
	s.debug("SIGTERM sent successfully; waiting for waitErr")
	if err := <-waitErr; err != nil {
		s.debug("waitErr not nil: %s", err)
		return err
	}
	s.debug("exited successfully")

	return nil
}

// DumpTail prints out the last n lines of combined (stdout + stderr) output
// from this service.
func (s *Service) DumpTail(t *testing.T, n int) {
	path := filepath.Join(s.LogDir, "combined")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		rtLog("ERROR unable to read log file %s: %s", path, err)
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) < n {
		n = len(lines)
	}
	out := strings.Join(lines[len(lines)-n:], "\n") + "\n"
	prefix := fmt.Sprintf("%s:%s:combined> ", t.Name(), s.InstanceName)
	outPipe, err := prefixedPipe(prefix)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(outPipe, out)
}
