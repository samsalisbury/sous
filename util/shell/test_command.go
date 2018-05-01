package shell

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/nyarly/spies"
)

type (
	// TestCommand is a test wrapper for Command
	TestCommand struct {
		*spies.Spy
	}

	// TestCommandController Allows an associated TestCommand to be controlled and inspected
	TestCommandController struct {
		cmd *TestCommand
		*spies.Spy
	}

	// TestCommandResult describes the dummy results of a dummy command
	TestCommandResult struct {
		SO, SE []byte
		Err    error
		Status int
	}
)

// NewTestCommand returns a new TestCommand and a TestCommandController with
// which it can be controlled and inspected.
func NewTestCommand() (*TestCommand, *TestCommandController) {
	spy := spies.NewSpy()
	cmd := &TestCommand{spy}
	return cmd, &TestCommandController{cmd, spy}
}

// Stdout implements Cmd on TestCommand
func (c *TestCommand) Stdout() (string, error) {
	res := c.Called()
	return res.String(0), res.Error(1)
}

// Stderr implements Cmd on TestCommand
func (c *TestCommand) Stderr() (string, error) {
	res := c.Called()
	return res.String(0), res.Error(1)
}

// SetStdin implements Cmd on TestCommand
func (c *TestCommand) SetStdin(r io.Reader) {
	c.Called(r)
}

// Lines implements Cmd on TestCommand
func (c *TestCommand) Lines() ([]string, error) {
	res := c.Called()
	return res.Get(0).([]string), res.Error(1)
}

// Table implements Cmd on TestCommand
func (c *TestCommand) Table() ([][]string, error) {
	res := c.Called()
	return res.Get(0).([][]string), res.Error(1)
}

// JSON implements Cmd on TestCommand
func (c *TestCommand) JSON(v interface{}) error {
	res := c.Called()

	err := json.Unmarshal([]byte(res.String(0)), v)
	if err != nil {
		panic(err)
	}
	return res.Error(1)
}

// ExitCode implements Cmd on TestCommand
func (c *TestCommand) ExitCode() (int, error) {
	res := c.Called()
	return res.Int(0), res.Error(1)
}

// Result implements Cmd on TestCommand
func (c *TestCommand) Result() (*Result, error) {
	res := c.Called()
	return res.Get(0).(*Result), res.Error(1)
}

// SucceedResult implements Cmd on TestCommand
func (c *TestCommand) SucceedResult() (*Result, error) {
	res := c.Called()
	return res.Get(0).(*Result), res.Error(1)
}

// Succeed implements Cmd on TestCommand
func (c *TestCommand) Succeed() error {
	res := c.Called()
	return res.Error(0)
}

// Fail implements Cmd on TestCommand
func (c *TestCommand) Fail() error {
	res := c.Called()
	return res.Error(0)
}

// FailResult implements Cmd on TestCommand
func (c *TestCommand) FailResult() (*Result, error) {
	res := c.Called()
	return res.Get(0).(*Result), res.Error(1)
}

// String implements Cmd on TestCommand
func (c *TestCommand) String() string {
	res := c.Called()
	return res.String(0)
}

// ResultSuccess sets up the TestCommand to behave like it ran successfully with particular stdout/stderr.
func (c *TestCommandController) ResultSuccess(out, err string) {
	ob := &Output{bytes.NewBufferString(out)}
	eb := &Output{bytes.NewBufferString(err)}
	cb := &Output{bytes.NewBufferString(out + err)}
	res := &Result{Command: c.cmd, Stdout: ob, Stderr: eb, Combined: cb, Err: nil, ExitCode: 0}

	c.MatchMethod("Result", spies.AnyArgs, res, nil)
	c.MatchMethod("SucceedResult", spies.AnyArgs, res, nil)
	c.MatchMethod("Succeed", spies.AnyArgs, nil)
	c.MatchMethod("Stdout", spies.AnyArgs, out, nil)
	c.MatchMethod("Stderr", spies.AnyArgs, err, nil)
}

// ResultFailure see above but unsuccessfully
func (c *TestCommandController) ResultFailure(out, err string) {
	ob := &Output{bytes.NewBufferString(out)}
	eb := &Output{bytes.NewBufferString(err)}
	cb := &Output{bytes.NewBufferString(out + err)}
	newErr := errors.New(err)
	res := &Result{Command: c.cmd, Stdout: ob, Stderr: eb, Combined: cb, Err: newErr, ExitCode: 1}

	c.MatchMethod("Result", spies.AnyArgs, res, nil)
	c.MatchMethod("SucceedResult", spies.AnyArgs, res, newErr)
	c.MatchMethod("Succeed", spies.AnyArgs, newErr)
	c.MatchMethod("Stdout", spies.AnyArgs, out, nil)
	c.MatchMethod("Stderr", spies.AnyArgs, err, nil)
}
