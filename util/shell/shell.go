// Package shell provides convenience wrappers around os/exec.
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
	"io"
	"os"
)

type (
	// Shell is a shell helper.
	Shell interface {
		// Clone returns a deep copy of this shell.
		Clone() Shell
		// Dir simply returns the current working directory for this Shell
		Dir() string
		// List returns all files (including dotfiles) inside Dir.
		List() ([]os.FileInfo, error)
		// Abs returns the absolute path of the path provided in relation to this shell.
		// If the path is already absolute, it is returned simplified but otherwise
		// unchanged.
		Abs(path string) string
		// Stat calls os.Stat on the path provided, relative to the current
		// shell's working directory.
		Stat(path string) (os.FileInfo, error)
		// Exists returns true if the path definitely exists. It swallows
		// any errors and returns false, in the case that e.g. permissions
		// prevent the check from working correctly.
		Exists(path string) bool
		// ConsoleEcho outputs the line as if it were typed at the console
		ConsoleEcho(line string)
		// LongRunning marks a Shell as dealing with long running commands
		LongRunning(bool)
		// Cmd creates a new Command based on this shell.
		Cmd(name string, args ...interface{}) Cmd
		// CD changes the directory of this shell to the path specified. If the path is
		// relative, the directory is attempted to be changed relative to the current
		// dir. If the directory does not exist, CD returns an error.
		CD(dir string) error
		// Run(...) is a shortcut for shell.Cmd(...).Succeed()
		Run(name string, args ...interface{}) error
		// Stdout(...) is a shortcut for shell.Cmd(...).Stdout()
		Stdout(name string, args ...interface{}) (string, error)
		// Stderr(...) is a shortcut for shell.Cmd(...).Stderr()
		Stderr(name string, args ...interface{}) (string, error)
		// ExitCode(...) is a shortcut for shell.Cmd(...).ExitCode()
		ExitCode(name string, args ...interface{}) (int, error)
		// Lines(...) is a shortcut for shell.Cmd(...).Lines()
		Lines(name string, args ...interface{}) ([]string, error)
		// JSON(x, ...) is a shortcut for shell.Cmd(...).JSON(x)
		JSON(v interface{}, name string, args ...interface{}) error
	}

	// Cmd is an interface to describe commands being executed by a Shell
	Cmd interface {
		// Stdout returns the stdout stream as a string. It returns an error for the
		// same reasons as .Succeed
		Stdout() (string, error)
		// Stderr is returns the stderr stream as a string. It returns an error for the
		// same reasons as .Result
		Stderr() (string, error)
		// Stdin sets the Stdin of the command
		SetStdin(io.Reader)
		// Lines returns Stdout split by newline. Leading and trailing empty lines are
		// removed, and each line is trimmed of whitespace.
		Lines() ([]string, error)
		// Table is similar to Lines, except lines are further split by whitespace.
		Table() ([][]string, error)
		// JSON tries to parse the stdout from the command as JSON, populating the
		// value you pass. (This value should be a pointer.)
		JSON(v interface{}) error
		// ExitCode only returns an error if there were io issues starting the command,
		// it does not return an error for a command which fails and returns an error
		// code, which is unlike most of Sh's methods. If it returns an error, then it
		// also returns -1 for the exit code.
		ExitCode() (int, error)
		// Result only returns an error if it's a startup error, not if the command
		// itself exits with an error code. If you need an error to be returned on
		// non-zero exit codes, use SucceedResult instead.
		Result() (*Result, error)
		// SucceedResult is similar to Result, except that it also returns an error if
		// the command itself fails (returns a non-zero exit code).
		SucceedResult() (*Result, error)
		// Succeed returns an error if the command fails for any reason (fails to start
		// or finishes with a non-zero exist code).
		Succeed() error
		// Fail returns an error if the command succeeds to execute, or if it fails to
		// start. It returns nil only if the command starts successfully and then exits
		// with a non-zero exit code.
		Fail() error
		// FailResult returns an error when the command fails to be invoked at all, or
		// when the command is successfully invoked, and then runs successfully. It
		// does not return an error when the command is invoked successfully and then
		// fails.
		FailResult() (*Result, error)
		// String prints out this command.
		String() string
	}
)
