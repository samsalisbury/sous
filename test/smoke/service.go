package smoke

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// Service represents a binary that is intended to be run as a long-running
// process. It includes things like crash detection.
type Service struct {
	Bin
	Proc *os.Process
}

// NewService returns a new service bound to the provided binary.
func NewService(bin Bin) *Service {
	return &Service{Bin: bin}
}

// Start starts this service.
func (s *Service) Start(t *testing.T, subcmd string, flags Flags, args ...string) {
	t.Helper()

	prepared := s.Command(t, subcmd, flags, args...)

	cmd := prepared.Cmd
	if err := cmd.Start(); err != nil {
		t.Fatalf("error starting server %q: %s", s.InstanceName, err)
	}

	if cmd.Process == nil {
		panic("cmd.Process nil after cmd.Start")
	}

	go func() {
		id := fmt.Sprintf("%s:%s", t.Name(), s.InstanceName)

		var ps *os.ProcessState
		select {
		// In this case the process ended before the test finished.
		case err := <-func() <-chan error {
			var err error
			c := make(chan error, 1)
			go func() {
				ps, err = cmd.Process.Wait()
				c <- err
			}()
			return c
		}():
			if err != nil {
				exitCode := tryGetExitCode("Wait error", err)
				rtLog("SERVER CRASHED: %s: %s (exit code %d); process state: %s", id, err, exitCode, ps)
				return
			}

			exitCode := tryGetExitCode("ps.Sys()", ps.Sys())

			if !ps.Exited() {
				// NOTE SS: This condition should not be possible, since after
				// calling Wait, the process should have exited. But it hasn't.
				// Even though 'should be impossible', this does happen in
				// practice.
				rtLog("OK: SERVER DID NOT EXIT: %s", id)
				return
			}
			if ps.Success() {
				rtLog("SERVER STOPPED: %s (exit code 0)", id)
			}
			// TODO SS: Dump log tail here as well for analysis.
			rtLog("SERVER CRASHED: exit code %d: %s; logs follow:", exitCode, id)
			s.DumpTail(t, 25)
		// In this case the process is still running.
		case <-s.TestFinished:
			rtLog("OK: SERVER STILL RUNNING AFTER TEST %s", id)
			// Do nothing.
		}
	}()

	s.Proc = cmd.Process
	writePID(t, s.Proc.Pid)
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
	out := strings.Join(lines[len(lines)-n:], "\n") + "\n"
	prefix := fmt.Sprintf("%s:%s:combined> ", t.Name(), s.InstanceName)
	outPipe, err := prefixedPipe(prefix)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(outPipe, out)
}
