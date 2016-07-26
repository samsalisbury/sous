package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/opentable/sous/util/whitespace"

	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/mapfs"
)

type (
	// TestShell is a test instance for Sh
	TestShell struct {
		Sh
		CmdsF   cmdGenerator
		FS      vfs.FileSystem
		History []*DummyCommand
	}

	cmdGenerator func(name string, args []interface{}) *DummyResult

	// DummyCommand is a test wrapper for Command
	DummyCommand struct {
		siStr *string
		Command
		DummyResult
	}

	// DummyResult describes the dummy results of a dummy command
	DummyResult struct {
		SO, SE []byte
		Err    error
		Status int
	}
)

func blissfulSuccess(n string, as []interface{}) *DummyResult {
	return nil
}

// NewTestShell builds a test shell, notionally at path, with a set of files available
// to it built from the files map. Note that relative paths in the map are
// considered wrt the path
func NewTestShell(path string, files map[string]string) (*TestShell, error) {
	ts := TestShell{
		CmdsF: blissfulSuccess,
		Sh: Sh{
			Cwd: path,
			Env: os.Environ(),
		},
	}
	fs := make(map[string]string)
	for n, c := range files {
		fs[strings.TrimPrefix(ts.Abs(n), "/")] = c
	}
	ts.FS = mapfs.New(fs)
	return &ts, nil
}

// Cmd creates a new Command based on this shell.
func (s *TestShell) Cmd(name string, args ...interface{}) Cmd {
	sargs := make([]string, len(args))
	for i, a := range args {
		sargs[i] = fmt.Sprint(a)
	}
	r := s.CmdsF(name, args)
	if r == nil {
		r = &DummyResult{
			SO:     []byte{},
			SE:     []byte{},
			Err:    nil,
			Status: 0,
		}
	}
	dc := &DummyCommand{
		DummyResult: *r,
		Command: Command{
			Dir:  s.Dir(), // is therefore the live shell...
			Name: name,
			Args: sargs,
		},
	}
	s.History = append(s.History, dc)
	return dc
}

// List returns all files (including dotfiles) inside Dir.
func (s *TestShell) List() ([]os.FileInfo, error) {
	ios, err := s.FS.ReadDir(s.Dir())
	if err != nil {
		return nil, err
	}
	for i := range ios {
		if ios[i].Name() == "__exists__" {
			ios[i] = ios[len(ios)-1]
			return ios[:len(ios)-1], nil
		}
	}

	return ios, err
}

// Exists returns true if the path definitely exists. It swallows
// any errors and returns false, in the case that e.g. permissions
// prevent the check from working correctly.
func (s *TestShell) Exists(path string) bool {
	_, err := s.Stat(path)
	return err == nil
}

// Stat calls os.Stat on the path provided, relative to the current
// shell's working directory.
func (s *TestShell) Stat(path string) (os.FileInfo, error) {
	return s.FS.Stat(s.Abs(path))
}

// SetStdin sets the stdin on the command
func (c *DummyCommand) SetStdin(in io.Reader) {
	c.Stdin = in
}

// StdinString returns the Stdin provided for this command as a string
func (c *DummyCommand) StdinString() string {
	if c.Stdin == nil {
		return ""
	}
	if c.siStr != nil {
		return *c.siStr
	}
	bf := &bytes.Buffer{}
	io.Copy(bf, c.Stdin)
	s := bf.String()
	c.siStr = &s
	return *c.siStr
}

// Stdout returns the stdout stream as a string. It returns an error for the
// same reasons as Succeed.
func (c *DummyCommand) Stdout() (string, error) {
	r, err := c.SucceedResult()
	if err != nil {
		return "", err
	}
	return r.Stdout.String(), nil
}

// Stderr is returns the stderr stream as a string. It returns an error for the
// same reasons as Result.
func (c *DummyCommand) Stderr() (string, error) {
	r, err := c.Result()
	if err != nil {
		return "", err
	}
	return r.Stderr.String(), nil
}

// Lines returns Stdout split by newline. Leading and trailing empty lines are
// removed, and each line is trimmed of whitespace.
func (c *DummyCommand) Lines() ([]string, error) {
	stdout, err := c.Stdout()
	if err != nil {
		return nil, err
	}
	rawLines := strings.Split(stdout, "\n")
	lines := []string{}
	for _, l := range rawLines {
		trimmed := whitespace.Trim(l)
		if len(trimmed) == 0 {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines, nil
}

// Table is similar to Lines, except lines are further split by whitespace.
func (c *DummyCommand) Table() ([][]string, error) {
	r, err := c.SucceedResult()
	if err != nil {
		return nil, err
	}
	return r.Stdout.Table(), nil
}

// JSON tries to parse the stdout from the command as JSON, populating the
// value you pass. (This value should be a pointer.)
func (c *DummyCommand) JSON(v interface{}) error {
	r, err := c.Result()
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(r.Stdout.Reader())
	return decoder.Decode(v)
}

// ExitCode only returns an error if there were io issues starting the command,
// it does not return an error for a command which fails and returns an error
// code, which is unlike most of Sh's methods. If it returns an error, then it
// also returns -1 for the exit code.
func (c *DummyCommand) ExitCode() (int, error) {
	r, err := c.Result()
	if err != nil {
		return -1, err
	}
	return r.ExitCode, nil
}

// Result attempts to simulate the running of a command by filling the
// appropriate buffers from its Test fields and forwarding its Test exit codes
func (c *DummyCommand) Result() (*Result, error) {
	//command := exec.Command(c.Name, c.Args...)
	outbuf := &bytes.Buffer{}
	errbuf := &bytes.Buffer{}
	combinedbuf := &bytes.Buffer{}
	outWriters := []io.Writer{outbuf, combinedbuf}
	errWriters := []io.Writer{errbuf, combinedbuf}
	if c.Command.TeeOut != nil {
		outWriters = append(outWriters, c.TeeOut)
	}
	if c.Command.TeeErr != nil {
		errWriters = append(errWriters, c.TeeErr)
	}

	outWr := io.MultiWriter(outWriters...)
	errWr := io.MultiWriter(errWriters...)

	outWr.Write(c.DummyResult.SO)
	errWr.Write(c.DummyResult.SE)

	return &Result{
		Command:  &c.Command, // XXX thus the live command...
		Stdout:   &Output{outbuf},
		Stderr:   &Output{errbuf},
		Combined: &Output{combinedbuf},
		Err:      c.DummyResult.Err,
		ExitCode: c.DummyResult.Status,
	}, nil
}

// SucceedResult is similar to Result, except that it also returns an error if
// the command itself fails (returns a non-zero exit code).
func (c *DummyCommand) SucceedResult() (*Result, error) {
	r, err := c.Result()
	if err != nil {
		return r, err
	}
	if r.Err != nil {
		return r, newError(r.Err, r)
	}
	return r, nil
}

// Succeed returns an error if the command fails for any reason (fails to start
// or finishes with a non-zero exist code).
func (c *DummyCommand) Succeed() error {
	_, err := c.SucceedResult()
	return err
}

// Fail returns an error if the command succeeds to execute, or if it fails to
// start. It returns nil only if the command starts successfully and then exits
// with a non-zero exit code.
func (c *DummyCommand) Fail() error {
	_, err := c.FailResult()
	return err
}

// FailResult returns an error when the command fails to be invoked at all, or
// when the command is successfully invoked, and then runs successfully. It
// does not return an error when the command is invoked successfully and then
// fails.
func (c *DummyCommand) FailResult() (*Result, error) {
	r, err := c.Result()
	if err != nil {
		return r, err
	}
	if r.Err == nil {
		return r, fmt.Errorf("command %q succeeded, expected failure", c)
	}
	return r, nil
}

func (c *DummyCommand) String() string {
	args := strings.Join(c.Command.Args, " ")
	// using shell comment token
	return fmt.Sprintf("%s %s # the dummy version", c.Command.Name, args)
}

// Run (...) is a shortcut for shell.Cmd(...).Succeed()
func (s *TestShell) Run(name string, args ...interface{}) error {
	return s.Cmd(name, args...).Succeed()
}

// Stdout (...) is a shortcut for shell.Cmd(...).Stdout()
func (s *TestShell) Stdout(name string, args ...interface{}) (string, error) {
	return s.Cmd(name, args...).Stdout()
}

// Stderr (...) is a shortcut for shell.Cmd(...).Stderr()
func (s *TestShell) Stderr(name string, args ...interface{}) (string, error) {
	return s.Cmd(name, args...).Stderr()
}

// ExitCode (...) is a shortcut for shell.Cmd(...).ExitCode()
func (s *TestShell) ExitCode(name string, args ...interface{}) (int, error) {
	return s.Cmd(name, args...).ExitCode()
}

// Lines (...) is a shortcut for shell.Cmd(...).Lines()
func (s *TestShell) Lines(name string, args ...interface{}) ([]string, error) {
	return s.Cmd(name, args...).Lines()
}

// JSON (x, ...) is a shortcut for shell.Cmd(...).JSON(x)
func (s *TestShell) JSON(v interface{}, name string, args ...interface{}) error {
	return s.Cmd(name, args...).JSON(v)
}
