package shell

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type (
	// Sh is a shel, and provides command defaults.
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

// Clone returns a deep copy of this shell.
func (s *Sh) Clone() *Sh {
	cp := *s
	cp.Env = make([]string, len(s.Env))
	copy(cp.Env, s.Env)
	cp.MonitorFuncs = make([]func(string, []string), len(s.MonitorFuncs))
	copy(cp.MonitorFuncs, s.MonitorFuncs)
	return &cp
}

// CD changes the directory of this shell to the path specified. If the path i
// relative, the directory is attempted to be changed from Dir to Dir + the
// relative directory. If the directory does not exist, CD returns an error.
func (s *Sh) CD(dir string) error {
	if !filepath.IsAbs(dir) {
		dir = filepath.Clean(filepath.Join(s.Dir, dir))
	}
	f, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !f.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	s.Dir = dir
	return nil
}

// Cmd creates a new Command with the passed name and args, using the default
// shell.
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

func (s *Sh) Stdout(name string, args ...interface{}) (string, error) {
	return s.Cmd(name, args...).Stdout()
}
func (s *Sh) Stderr(name string, args ...interface{}) (string, error) {
	return s.Cmd(name, args...).Stderr()
}
