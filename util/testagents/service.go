package testagents

import (
	"fmt"
	"io/ioutil"
	"log"
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
	waitResult      error
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
	prepared, err := s.Command(nil, subcmd, flags, args...)
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
	s.ProcMan.WritePID(t, s.Cmd.Process.Pid)

	go s.wait(t)
}

// wait waits for either test to finish or the service to exit.
// It's a race, we hope that the test finishes first.
// If not, we dump logs, exit code and  some other info to help determine the
// cause of the early exit.
func (s *Service) wait(t *testing.T) bool {
	defer close(s.ReadyForCleanup)
	s.debug("[detect early exit] waiting for test to finish or process to exit early")
	select {
	// In this case the process ended before the test finished.
	case waitErr := <-s.waitChan():
		safeClose(s.Stopped) // Avoid trying to clean up later.
		s.debug("[detect early exit] got wait error: %v", waitErr)
		exitCode, exitCodeErr := exitCode(waitErr)
		if exitCodeErr != nil {
			s.LogFunc("[ERROR:%s]: [detect early exit] unable to determine exit code: %s", s.ID(), exitCodeErr)
		} else {
			s.debug("[detect early exit] exited with code %d; error: %v; logs follow:", exitCode, exitCodeErr)
		}
		s.DumpTail(3)
		return true
	case <-s.Stopped:
		s.debug("[detect early exit] ok - did not exit prematurely")
		return false
	}
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

func exitCode(waitErr error) (int, error) {
	if waitErr == nil {
		return 0, nil
	}
	ee, ok := waitErr.(*exec.ExitError)
	if !ok {
		return -2, fmt.Errorf("wait failed, returned a %T: %s", waitErr, waitErr)
	}
	ws, ok := ee.Sys().(interface {
		ExitStatus() int
	})
	if !ok {
		return -3, fmt.Errorf("expected *exec.ExitError.Sys() to return a type with method ExitStatus() int, but got a %T; maybe there is a breaking change in stdlib? Please report this to the maintainer", ee.Sys())
	}

	if ws.ExitStatus() > 0 {
		return ws.ExitStatus(), nil
	}

	return ws.ExitStatus(), fmt.Errorf("invalid failure exit status %d", ws.ExitStatus())
}

// waitChan returns a channel that sends one copy of the result of waiting on
// this process, and then is closed. It is safe to call waitChan concurrently
// and only one wait will actually be done on the process.
func (s *Service) waitChan() <-chan error {
	s.waitChanMu.Lock()
	defer s.waitChanMu.Unlock()
	s.waitOnce.Do(func() {
		go func() {
			waitErr := s.Cmd.Wait()
			exitCode, exitCodeErr := exitCode(waitErr)
			if exitCodeErr != nil {
				s.LogFunc("[ERROR:%s] Ignoring wait error: %s; unable to determine exit code: %s",
					s.ID(), waitErr, exitCodeErr)
			} else if !s.Cmd.ProcessState.Exited() {
				log.Panicf("[PANIC:%s] process not exited after wait, but reports exit code %d",
					s.ID(), exitCode)
			}
			s.waitResult = waitErr
			close(s.doneWaiting)
		}()
	})
	c := make(chan error, 1)
	go func() {
		defer close(c)
		<-s.doneWaiting
		c <- s.waitResult
	}()
	return c
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
	s.LogFunc("[DEBUG:"+s.ID()+";pid:"+pid+"] "+f, a...)
}

// Stop stops this service.
func (s *Service) Stop() error {
	if closed(s.Stopped) {
		s.debug("not stopping as already stopped")
		return nil
	}
	safeClose(s.Stopped)
	if s.Cmd == nil || s.Cmd.Process == nil {
		s.debug("process is nil; cannot stop")
		return fmt.Errorf("cannot stop %s (not started)", s.InstanceName)
	}
	s.debug("process exists; waiting for exit")
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
	if waitErr := <-s.waitChan(); waitErr != nil {
		s.debug("waitErr not nil: %s", waitErr)
		return waitErr
	}
	s.debug("exited successfully")
	return nil
}

// DumpTail prints out the last n lines of combined (stdout + stderr) output
// from this service.
func (s *Service) DumpTail(n int) {
	path := filepath.Join(s.LogDir, "combined")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		s.LogFunc("ERROR unable to read log file %s: %s", path, err)
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) < n {
		n = len(lines)
	}
	out := strings.Join(lines[len(lines)-n:], "\n") + "\n"

	blockStart := fmt.Sprintf("BEGIN LOG DUMP (%d LINES)\n", n)
	blockEnd := "END LOG DUMP\n"
	s.prefixPrintf("logs/combined", blockStart+out+blockEnd)
}
