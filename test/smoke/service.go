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
)

// Service represents a binary that is intended to be run as a long-running
// process. It includes things like crash detection.
type Service struct {
	Bin
	Proc            *os.Process
	ReadyForCleanup chan struct{}
}

// NewService returns a new service bound to the provided binary.
func NewService(bin Bin) *Service {
	bin.ShouldStillBeRunningAfterTest = true
	return &Service{
		Bin:             bin,
		ReadyForCleanup: make(chan struct{}),
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
		s.detectPrematureExit(t, cmd)
		close(s.ReadyForCleanup)
	}()
}

func (s *Service) detectPrematureExit(t *testing.T, cmd *exec.Cmd) {
	id := s.ID()
	select {
	// In this case the process ended before the test finished.
	case wr := <-s.waitChan(cmd):
		rtLog("SERVER CRASHED (pid %d): exit code %d: %s; logs follow:", s.Proc.Pid,
			wr.exitCode(), id)
		s.DumpTail(t, 3)
		rtLog("END SERVER CRASH LOG (pid %d)", s.Proc.Pid)
	// In this case the process is still running.
	case <-s.TestFinished:
		// OK, test finished before this process exited.
	}
}

type waitResult struct {
	err error
	ps  *os.ProcessState
}

func (wr *waitResult) exitCode() int {
	err, ps := wr.err, wr.ps
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

// WaitChan returns a channel that sends the error from os.Process.Waiting on
// this command.
func (s *Service) waitChan(cmd *exec.Cmd) <-chan waitResult {
	var wr waitResult
	c := make(chan waitResult, 1)
	go func() {
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
	return -1
}

// Stop stops this service.
func (s *Service) Stop() error {
	<-s.ReadyForCleanup
	if s.Proc == nil {
		return fmt.Errorf("cannot stop %s (not started)", s.InstanceName)
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
