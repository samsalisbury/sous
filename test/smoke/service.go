package smoke

import (
	"fmt"
	"io/ioutil"
	"os"
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
	Proc            *os.Process
	ReadyForCleanup chan struct{}
	Stopped         chan struct{}
	waitOnce        sync.Once
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

	cmd := prepared.Cmd
	if err := cmd.Start(); err != nil {
		t.Fatalf("error starting %q: %s", s.InstanceName, err)
	}

	if cmd.Process == nil {
		t.Fatalf("[ERROR:%s] cmd.Process nil after cmd.Start", s.ID())
	}
	s.Proc = cmd.Process
	writePID(t, s.Proc.Pid)

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
		if exitCode >= 0 {
			s.debug("exited with code %d; error: %v; logs follow:", exitCode, err)
			s.DumpTail(t, 3)
			s.debug("end logs")
			safeClose(s.Stopped)
			return true
		}
		if count > 0 {
			time.Sleep(time.Second)
			s.debug("bad exit code %d; trying again", exitCode)
			return s.wait(t, count-1, true)
		}
		s.debug("bad exit code %d; no more retries", exitCode)
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
	err, ps := wr.err, wr.ps
	exitCode := tryGetExitCode("ps.Sys()", ps.Sys())
	if !ps.Exited() {
		return exitCode, fmt.Errorf("not exited; %s", wr)
	}
	if err != nil {
		return exitCode, fmt.Errorf("exited; %s", wr)
	}
	return exitCode, nil
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
	s.waitOnce.Do(func() {
		go func() {
			var wr waitResult
			wr.ps, wr.err = s.Proc.Wait()
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
	if s.Proc != nil {
		pid = strconv.Itoa(s.Proc.Pid)
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
	if s.Proc == nil {
		s.debug("process is nil; cannot stop")
		return fmt.Errorf("cannot stop %s (not started)", s.InstanceName)
	}
	s.debug("got process")
	waitErr := make(chan error, 1)
	var ps *os.ProcessState
	go func() {
		defer close(waitErr)
		var err error
		s.debug("waiting for exit")
		wr := <-s.waitChan()
		ps, err = wr.ps, wr.err
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
	if err := s.Proc.Signal(syscall.SIGTERM); err != nil {
		s.debug("error sending SIGTERM: %s", err)
		return fmt.Errorf("sending SIGTERM: %s", err)
	}
	s.debug("SIGTERM sent successfully; waiting for waitErr")
	if err := <-waitErr; err != nil {
		s.debug("waitErr not nil: %s", err)
		return err
	}
	s.debug("waitErr is nil")
	if !ps.Exited() {
		s.debug("still not exited, trying kill")
		if err := s.Proc.Kill(); err != nil {
			s.debug("kill failed: %s", err)
			return fmt.Errorf("cannot kill %s: %s", s.InstanceName, err)
		}
		s.debug("kill succeeded")
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
