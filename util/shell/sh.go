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
		// Cwd is the working directory of the shell.
		Cwd string
		// Env is the environment variables of the shell.
		Env []string
		// If TeeEcho is non-nil, all the commands executed on this shell will be
		// written to it
		TeeEcho,
		// If TeeOut is non-nil, then all stdout commands get written to it, in
		// addition to being preserved in the Result.
		TeeOut,
		// TeeErr is similar to TeeOut, except that it has stderr written to it
		// instead of stdout.
		TeeErr io.Writer
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
		Cwd: wd,
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

// Dir returns the directory for this shell
func (s *Sh) Dir() string {
	return s.Cwd
}

// Clone returns a deep copy of this shell.
func (s *Sh) Clone() Shell {
	return s.clone()
}

func (s *Sh) clone() *Sh {
	cp := *s
	cp.Env = make([]string, len(s.Env))
	copy(cp.Env, s.Env)
	return &cp
}

// CD changes the directory of this shell to the path specified. If the path is
// relative, the directory is attempted to be changed relative to the current
// dir. If the directory does not exist, CD returns an error.
func (s *Sh) CD(dir string) error {
	if !filepath.IsAbs(dir) {
		dir = filepath.Clean(filepath.Join(s.Cwd, dir))
	}
	s.Cwd = dir
	f, err := os.Stat(s.Cwd)
	if err != nil {
		return err
	}
	if !f.IsDir() {
		return fmt.Errorf("%s is not a directory", s.Cwd)
	}
	return nil
}

// Cmd creates a new Command based on this shell.
func (s *Sh) Cmd(name string, args ...interface{}) Cmd {
	sargs := make([]string, len(args))
	for i, a := range args {
		sargs[i] = fmt.Sprint(a)
	}
	return &Command{
		Sh:   *s.clone(),
		Name: name,
		Args: sargs,
	}
}

// ConsoleEcho prints the command that sous is executing out
func (s *Sh) ConsoleEcho(line string) {
	if s.TeeEcho != nil {
		s.TeeEcho.Write([]byte(fmt.Sprintf("  (Sous)> %s\n", line)))
	}
}

// List returns all files (including dotfiles) inside Dir.
func (s *Sh) List() ([]os.FileInfo, error) {
	return ioutil.ReadDir(s.Cwd)
}

// Exists returns true if the path definitely exists. It swallows
// any errors and returns false, in the case that e.g. permissions
// prevent the check from working correctly.
func (s *Sh) Exists(path string) bool {
	_, err := s.Stat(path)
	return err == nil
}

// Stat calls os.Stat on the path provided, relative to the current
// shell's working directory.
func (s *Sh) Stat(path string) (os.FileInfo, error) {
	return os.Stat(s.Abs(path))
}

// Abs returns the absolute path of the path provided in relation to this shell.
// If the path is already absolute, it is returned simplified but otherwise
// unchanged.
func (s *Sh) Abs(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(s.Cwd, path)
}

// Run (...) is a shortcut for shell.Cmd(...).Succeed()
func (s *Sh) Run(name string, args ...interface{}) error {
	return s.Cmd(name, args...).Succeed()
}

// Stdout (...) is a shortcut for shell.Cmd(...).Stdout()
func (s *Sh) Stdout(name string, args ...interface{}) (string, error) {
	return s.Cmd(name, args...).Stdout()
}

// Stderr (...) is a shortcut for shell.Cmd(...).Stderr()
func (s *Sh) Stderr(name string, args ...interface{}) (string, error) {
	return s.Cmd(name, args...).Stderr()
}

// ExitCode (...) is a shortcut for shell.Cmd(...).ExitCode()
func (s *Sh) ExitCode(name string, args ...interface{}) (int, error) {
	return s.Cmd(name, args...).ExitCode()
}

// Lines (...) is a shortcut for shell.Cmd(...).Lines()
func (s *Sh) Lines(name string, args ...interface{}) ([]string, error) {
	return s.Cmd(name, args...).Lines()
}

// JSON (x, ...) is a shortcut for shell.Cmd(...).JSON(x)
func (s *Sh) JSON(v interface{}, name string, args ...interface{}) error {
	return s.Cmd(name, args...).JSON(v)
}
