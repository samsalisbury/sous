package shell

import (
	"os"

	"github.com/opentable/sous/util/spies"
	"github.com/stretchr/testify/mock"
)

type (
	// TestShell is a test instance for Sh
	TestShell struct {
		*spies.Spy
		/*
			Sh
			CmdsF   cmdGenerator
			FS      vfs.FileSystem
			History []*DummyCommand
		*/
	}

	TestShellController struct {
		*spies.Spy
	}
)

//Clone implements Shell on TestShell
func (s *TestShell) Clone() Shell {
	s.Called()
	return s
}

//Dir implements Shell on TestShell
func (s *TestShell) Dir() string {
	res := s.Called()
	return res.String(0)
}

//Abs implements Shell on TestShell
func (s *TestShell) Abs(path string) string {
	res := s.Called(path)
	return res.String(0)
}

//ConsoleEcho implements Shell on TestShell
func (s *TestShell) ConsoleEcho(line string) {
	s.Called(line)
}

//LongRunning implements Shell on TestShell
func (s *TestShell) LongRunning(is bool) {
	s.Called(is)
}

//Dir implements Shell on TestShell
func (s *TestShell) CD(dir string) error {
	res := s.Called(dir)
	return res.Error(0)
}

// NewTestShell builds a test shell, notionally at path, with a set of files available
// to it built from the files map. Note that relative paths in the map are
// considered wrt the path
func NewTestShell() (*TestShell, *TestShellController) {
	spy := spies.NewSpy()
	return &TestShell{Spy: spy}, &TestShellController{Spy: spy}
}

// Cmd creates a new Command based on this shell.
func (s *TestShell) Cmd(name string, args ...interface{}) Cmd {
	res := s.Called(name, args)
	cmd, _ := NewTestCommand()
	return res.GetOr(0, cmd).(Cmd)
}

// List returns all files (including dotfiles) inside Dir.
func (s *TestShell) List() ([]os.FileInfo, error) {
	res := s.Called()
	return res.GetOr(0, []os.FileInfo{}).([]os.FileInfo), res.Error(1)
}

// Exists returns true if the path definitely exists. It swallows
// any errors and returns false, in the case that e.g. permissions
// prevent the check from working correctly.
func (s *TestShell) Exists(path string) bool {
	res := s.Called(path)
	return res.Bool(0)
}

// Stat calls os.Stat on the path provided, relative to the current
// shell's working directory.
func (s *TestShell) Stat(path string) (os.FileInfo, error) {
	res := s.Called(path)
	return res.GetOr(0, nil).(os.FileInfo), res.Error(1)
}

// Run (...) is a shortcut for shell.Cmd(...).Succeed()
func (s *TestShell) Run(name string, args ...interface{}) error {
	s.Called(name, args)
	return s.Cmd(name, args...).Succeed()
}

// Stdout (...) is a shortcut for shell.Cmd(...).Stdout()
func (s *TestShell) Stdout(name string, args ...interface{}) (string, error) {
	s.Called(name, args)
	return s.Cmd(name, args...).Stdout()
}

// Stderr (...) is a shortcut for shell.Cmd(...).Stderr()
func (s *TestShell) Stderr(name string, args ...interface{}) (string, error) {
	s.Called(name, args)
	return s.Cmd(name, args...).Stderr()
}

// ExitCode (...) is a shortcut for shell.Cmd(...).ExitCode()
func (s *TestShell) ExitCode(name string, args ...interface{}) (int, error) {
	s.Called(name, args)
	return s.Cmd(name, args...).ExitCode()
}

// Lines (...) is a shortcut for shell.Cmd(...).Lines()
func (s *TestShell) Lines(name string, args ...interface{}) ([]string, error) {
	s.Called(name, args)
	return s.Cmd(name, args...).Lines()
}

// JSON (x, ...) is a shortcut for shell.Cmd(...).JSON(x)
func (s *TestShell) JSON(v interface{}, name string, args ...interface{}) error {
	s.Called(name, args)
	return s.Cmd(name, args...).JSON(v)
}

func cmdWithArgs(parts ...string) func(string, mock.Arguments) bool {
	return func(method string, args mock.Arguments) bool {
		if method != "Cmd" {
			return false
		}
		if args.String(0) != parts[0] {
			return false
		}

		ai := args.Get(1).([]interface{})
		for n, part := range parts[1:] {
			if n >= len(ai) {
				return false
			}

			if ai[n] != part {
				return false
			}
		}
		return true
	}
}

func (ctl *TestShellController) CmdFor(parts ...string) (*TestCommand, *TestCommandController) {
	cmd, c := NewTestCommand()
	ctl.Match(cmdWithArgs(parts...), cmd)
	return cmd, c
}

func (ctl *TestShellController) CmdsLike(parts ...string) []spies.Call {
	return ctl.CallsMatching(cmdWithArgs(parts...))
}
