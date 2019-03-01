package testagents

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// PreparedCmd is a command ready to run.
type PreparedCmd struct {
	invocation
	Cmd      *exec.Cmd
	Cancel   func()
	PreRun   func() error
	PostRun  func() error
	executed *ExecutedCMD
}

func (c *PreparedCmd) tryGetExitCode() (int, error) {
	if c.Cmd == nil {
		return -1, fmt.Errorf("Cmd is nil")
	}
	if c.Cmd.ProcessState == nil {
		return -1, fmt.Errorf("command not run (no process state)")
	}
	if !c.Cmd.ProcessState.Exited() {
		return -1, fmt.Errorf("command not finished")
	}
	if c.Cmd.ProcessState.Sys() == nil {
		return -1, fmt.Errorf("process state returned nil for Sys()")
	}
	sys := c.Cmd.ProcessState.Sys()
	ws, ok := sys.(syscall.WaitStatus)
	if !ok {
		return -1, fmt.Errorf("process state Sys() was a %T; want a syscall.WaitStatus", sys)
	}
	es := ws.ExitStatus()
	if es < 0 {
		return es, fmt.Errorf("invalid negative exit status %d", es)
	}
	return es, nil
}

func (c *PreparedCmd) runWithTimeout(timeout time.Duration) error {
	defer c.PostRun()
	c.PreRun()
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		select {
		case errCh <- c.Cmd.Run():
		case <-time.After(timeout):
			errCh <- fmt.Errorf("command timed out after %s: %s", timeout, c)
			c.Cancel()
		}
	}()
	return <-errCh
}
