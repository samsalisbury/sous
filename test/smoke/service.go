package smoke

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
}

// NewService returns a new service bound to the provided binary.
func NewService(bin Bin) *Service {
	bin.ShouldStillBeRunningAfterTest = true
	return &Service{
		Bin:             bin,
		ReadyForCleanup: make(chan struct{}),
		Stopped:         make(chan struct{}),
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
		t.Fatalf("error starting server %q: %s", s.InstanceName, err)
	}

	if cmd.Process == nil {
		t.Fatal("cmd.Process nil after cmd.Start")
	}
	s.Proc = cmd.Process
	writePID(t, s.Proc.Pid)

	go func() {
		if s.exitedEarly(t, cmd, 10, false) {
			rtLog("NOT STOPPING CRASHED PROC: %s (pid %d)", s.ID(), s.Proc.Pid)
			return
		}
		close(s.ReadyForCleanup)
	}()
}

func debugLog(f string, a ...interface{}) {
	rtLog("[DEBUG] "+f, a...)
}

func (s *Service) exitedEarly(t *testing.T, cmd *exec.Cmd, count int, looping bool) bool {
	id := s.ID()
	if looping {
		select {
		default:
		case <-s.Finished.Finished:
			// OK, test finished before this process exited.
			debugLog("TEST FINISHED (EARLY CHECK) FOR PID %d", s.Proc.Pid)
			// return false
		}
	}

	select {
	// In this case the process ended before the test finished.
	case wr := <-s.waitChan(cmd):
		if wr.exitCode() < 1 && count > 0 {
			debugLog("FISHY EXIT CODE %d from pid %d; trying again %d time(s); %s", wr.exitCode(), wr.ps.Pid(), count-1, s.ID())
			time.Sleep(time.Second)
			return s.exitedEarly(t, cmd, count-1, true)
		}
		debugLog("SERVER CRASHED (pid %d): exit code %d: %s; logs follow:", s.Proc.Pid, wr.exitCode(), id)
		s.DumpTail(t, 3)
		debugLog("END SERVER CRASH LOG (pid %d)", s.Proc.Pid)
		safeClose(s.Stopped)
		return true
	// In this case the process is still running.
	case <-s.Finished.Failed:
		// OK, test finished before this process exited.
		debugLog("TEST FINISHED [FAILED] (LATE CHECK) FOR PID %d", s.Proc.Pid)
		if looping && count > 0 {
			return s.exitedEarly(t, cmd, count-1, true)
		}
		return false
	case <-s.Finished.Passed:
		// OK, test finished before this process exited.
		debugLog("TEST FINISHED [PASSED] (LATE CHECK) FOR PID %d", s.Proc.Pid)
		if looping && count > 0 {
			return s.exitedEarly(t, cmd, count-1, true)
		}
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

func (wr *waitResult) exitCode() int {
	err, ps := wr.err, wr.ps
	if !ps.Exited() {
		debugLog("[ERR] not exited: pid: %d; waitResult: %s", ps.Pid(), wr)
		return -2
	}
	debugLog("[OK] exited: pid: %d; waitResult: %s", ps.Pid(), wr)
	if err != nil {
		if exitCode := tryGetExitCode("Wait error", err); exitCode != -1 {
			return exitCode
		}
	}
	return tryGetExitCode("ps.Sys()", ps.Sys())
}

func (wr *waitResult) isCrash() bool {
	err, ps := wr.err, wr.ps
	if err != nil {
		return true
	}
	return err != nil || (ps.Exited() && !ps.Success())
}

// WaitChan returns a channel that sends the error from os.Process.Wait on
// this command.
func (s *Service) waitChan(cmd *exec.Cmd) <-chan waitResult {
	c := make(chan waitResult, 1)
	go func() {
		defer close(c)
		var wr waitResult
		wr.ps, wr.err = cmd.Process.Wait()
		c <- wr
	}()
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

// Stop stops this service.
func (s *Service) Stop() error {
	if closed(s.Stopped) {
		debugLog("[WAR] not stopping as already stopped")
		return nil
	}
	defer safeClose(s.Stopped)
	<-s.ReadyForCleanup
	if s.Proc == nil {
		return fmt.Errorf("cannot stop %s (not started)", s.InstanceName)
	}
	debugLog("Sending SIGTERM to pid %d", s.Proc.Pid)
	if err := s.Proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM: %s", err)
	}
	ps, err := s.Proc.Wait()
	if err != nil {
		return fmt.Errorf("wait failed: %s", err)
	}
	if !ps.Exited() {
		return fmt.Errorf("not stopped after wait")
	}

	if err := s.Proc.Kill(); err != nil {
		return fmt.Errorf("cannot kill %s: %s", s.InstanceName, err)
	}
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
