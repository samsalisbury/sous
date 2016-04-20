// The shell package provides convenience wrappers around os/exec.
//
// Specifically, it is designed to loosely emulate an ordinary shell session,
// with persistent directory context. It provides many helper functions around
// processing output streams into Go-friendly structures, and returning errors
// in an expected way.
//
// In general, and functions that specifically look for exit codes or output on
// stderr do not return an error for non-zero exit codes; they still return
// errors for other problems, like the process not starting due to failure to
// attach pipes, the binary not existing, etc. All other helper functions return
// errors for non-zero exit codes.
//
// This package is designed to aid with logging sessions, good for building CLI
// applications that shell out, and exposing these sessions to the user.
package shell

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type (
	// Sh is a shell helper.
	Sh struct {
		// Dir is the working directory of the shell.
		Dir string
		// Env is the environment variables of the shell.
		Env []string
		// If TeeOut is non-nil, then all stdout commands get written to it, in
		// addition to being preserved in the Result.
		TeeOut,
		// TeeErr is similar to TeeOut, except that it has stderr written to it
		// instead of stdout.
		TeeErr io.Writer
		// MonitorFuncs is a slice of funcs that are called for each command,
		// they are passed the command name, and a slice of args.
		MonitorFuncs []func(string, []string)
	}
)

// Default creates a new shell with all of the current environment
// variables from the current process added. This is useful if you want to,
// ensure that the PATH is set, along with all other environment variables
// the user had set when they invoked your Go program.
//
// If you do not need the user's environment, you can create a new shell also
// using &shell.Sh{}
func Default() (*Sh, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &Sh{
		Dir: wd,
		Env: os.Environ(),
	}, nil
}

// DefaultInDir is similar to Default, but immediately CDs into the specified
// directory. The path can be relative or absolute. If relative, it begins from
// the current working directory.
func DefaultInDir(path string) (*Sh, error) {
	sh := &Sh{Env: os.Environ()}
	return sh, sh.CD(path)
}

// Clone returns a deep copy of this shell.
func (s *Sh) Clone() *Sh {
	cp := *s
	cp.Env = make([]string, len(s.Env))
	copy(cp.Env, s.Env)
	cp.MonitorFuncs = make([]func(string, []string), len(s.MonitorFuncs))
	copy(cp.MonitorFuncs, s.MonitorFuncs)
	return &cp
}

// CD changes the directory of this shell to the path specified. If the path is
// relative, the directory is attempted to be changed relative to the current
// dir. If the directory does not exist, CD returns an error.
func (s *Sh) CD(dir string) error {
	if !filepath.IsAbs(dir) {
		dir = filepath.Clean(filepath.Join(s.Dir, dir))
	}
	s.Dir = dir
	f, err := os.Stat(s.Dir)
	if err != nil {
		return err
	}
	if !f.IsDir() {
		return fmt.Errorf("%s is not a directory", s.Dir)
	}
	return nil
}

// Cmd creates a new Command based on this shell.
func (s *Sh) Cmd(name string, args ...interface{}) *Command {
	sargs := make([]string, len(args))
	for i, a := range args {
		sargs[i] = fmt.Sprint(a)
	}
	return &Command{
		Sh:   s.Clone(),
		Name: name,
		Args: sargs,
	}
}

// List returns all files (including dotfiles) inside Dir.
func (s *Sh) List() ([]os.FileInfo, error) {
	return ioutil.ReadDir(s.Dir)
}

// Stdout(...) is a shortcut for shell.Cmd(...).Stdout()
func (s *Sh) Stdout(name string, args ...interface{}) (string, error) {
	return s.Cmd(name, args...).Stdout()
}

// Stderr(...) is a shortcut for shell.Cmd(...).Stderr()
func (s *Sh) Stderr(name string, args ...interface{}) (string, error) {
	return s.Cmd(name, args...).Stderr()
}

// ExitCode(...) is a shortcut for shell.Cmd(...).ExitCode()
func (s *Sh) ExitCode(name string, args ...interface{}) (int, error) {
	return s.Cmd(name, args...).ExitCode()
}

// Lines(...) is a shortcut for shell.Cmd(...).Lines()
func (s *Sh) Lines(name string, args ...interface{}) ([]string, error) {
	return s.Cmd(name, args...).Lines()
}

// JSON(x, ...) is a shortcut for shell.Cmd(...).JSON(x)
func (s *Sh) JSON(v interface{}, name string, args ...interface{}) error {
	return s.Cmd(name, args...).JSON(v)
}
