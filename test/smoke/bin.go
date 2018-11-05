package smoke

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/opentable/sous/util/filemap"
)

// Bin represents a binary under test.
type Bin struct {
	TestName string
	// BinPath is the absolute path to the executable file.
	BinPath string
	// BinName is the name of the executable file.
	BinName string
	// BaseDir is the root for logs/config files and other ancillary files.
	BaseDir   string
	ConfigDir string
	LogDir    string
	// InstanceName is used to identify this instance in test output.
	InstanceName string
	// RootDir is used to print current directory path in test output.
	// The printed path is Dir relative to RootDir.
	RootDir string

	// Dir is the working directory.
	Dir string
	// Env are persistent env vars to pass to invocations.
	Env map[string]string
	// MassageArgs is called on the total set of args passed to the command,
	// prior to execution; the args it returns are what is finally used.
	MassageArgs  func([]string) []string
	TestFinished <-chan struct{}

	// ShouldStillBeRunningAfterTest should is set to true for servers etc, it
	// enables crash detection.
	ShouldStillBeRunningAfterTest bool
}

// NewBin returns a new minimal Bin, all files will be created in subdirectories
// of baseDir.
func NewBin(t *testing.T, path, name, baseDir, rootDir string, finished <-chan struct{}) Bin {
	illegalChars := ":/>"
	if strings.ContainsAny(name, illegalChars) {
		log.Panicf("name %q contains at least one illegal character from %q", name, illegalChars)
	}
	binName := filepath.Base(path)
	return Bin{
		TestName:     t.Name(),
		BinPath:      path,
		BinName:      binName,
		BaseDir:      baseDir,
		ConfigDir:    filepath.Join(baseDir, "config"),
		LogDir:       filepath.Join(baseDir, "logs"),
		InstanceName: name,
		RootDir:      rootDir,
		Env:          map[string]string{},
		TestFinished: finished,
	}
}

// CD changes directory (accepts relative or absolute paths).
func (c *Bin) CD(path string) {
	if filepath.IsAbs(path) {
		c.Dir = path
		return
	}
	c.Dir = filepath.Clean(filepath.Join(c.Dir, path))
}

// ID returns the unique ID of this instance, formatted as:
// "test-name:instance-name".
func (c *Bin) ID() string {
	return fmt.Sprintf("%s:%s", c.TestName, c.InstanceName)
}

// Configure writes fm files relative to c.ConfigPath and ensures the log
// directory exists.
func (c *Bin) Configure(fms ...filemap.FileMap) error {
	if err := os.MkdirAll(c.ConfigDir, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(c.LogDir, os.ModePerm); err != nil {
		return err
	}
	fm := filemap.FileMap{}
	for _, f := range fms {
		fm = fm.Merge(f)
	}
	if err := fm.Write(c.ConfigDir); err != nil {
		return err
	}
	return nil
}

func flagArgs(f Flags) []string {
	if f == nil {
		return nil
	}
	m := f.FlagMap()
	p := f.FlagPrefix()
	names := make([]string, len(m))
	i := 0
	for name := range m {
		names[i] = name
		i++
	}
	sort.Strings(names)
	a := make([]string, 0, len(names)*2)
	for _, name := range names {
		a = append(a, p+name, m[name])
	}
	return a
}

// allArgs produces a []string representing all args determined by the sous
// subcommand, sous flags and any other args.
func (i invocation) allArgs() []string {
	all := strings.Split(i.subcmd, " ")
	all = append(all, flagArgs(i.flags)...)
	all = append(all, i.args...)
	return all
}

// Cmd generates an *exec.Cmd and cancellation func from final args.
func (c *Bin) cmd(finalArgs []string) (*exec.Cmd, context.CancelFunc) {
	cmd, cancel := mkCMD(c.Dir, c.BinPath, finalArgs...)
	for name, value := range c.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}
	return cmd, cancel
}

// Cmd generates an *exec.Cmd and cancellation func from an invocation.
func (c *Bin) Cmd(t *testing.T, i invocation) (*exec.Cmd, context.CancelFunc) {
	t.Helper()
	return c.cmd(i.finalArgs)
}

// Add quotes to args with spaces for printing.
func quotedArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.Contains(a, " ") {
			out[i] = `"` + a + `"`
		} else {
			out[i] = a
		}
	}
	return out
}

func quotedArgsString(args []string) string {
	return strings.Join(quotedArgs(args), " ")
}

// ExecutedCMD represents the reasult of a command having been run.
type ExecutedCMD struct {
	invocation
	finalArgs                []string
	Stdout, Stderr, Combined *bytes.Buffer
}

func newExecutedCMD(i invocation) *ExecutedCMD {
	return &ExecutedCMD{
		invocation: i,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Combined:   &bytes.Buffer{},
	}
}

// PreparedCmd is a command ready to run.
type PreparedCmd struct {
	invocation
	Cmd      *exec.Cmd
	Cancel   func()
	PreRun   func() error
	PostRun  func() error
	executed *ExecutedCMD
}

// Run runs the command.
func (c *Bin) Run(t *testing.T, subcmd string, f Flags, args ...string) (*ExecutedCMD, error) {
	cmd, err := c.Command(subcmd, f, args...)
	if err != nil {
		return nil, fmt.Errorf("setting up command failed: %s", err)
	}
	err = cmd.runWithTimeout(3 * time.Minute)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return cmd.executed, err
}

// Command returns the prepared command.
func (c *Bin) Command(subcmd string, f Flags, args ...string) (*PreparedCmd, error) {
	i := c.newInvocation(subcmd, f, args...)
	return c.configureCommand(i)
}

// invocation is the invocation directly from the test, without any formatting
// or manipulation.
type invocation struct {
	name, subcmd string
	flags        Flags
	args         []string
	finalArgs    []string
}

func (c *Bin) newInvocation(subcmd string, f Flags, args ...string) invocation {
	i := invocation{name: c.BinName, subcmd: subcmd, flags: f, args: args}
	i.finalArgs = i.allArgs()
	if c.MassageArgs != nil {
		i.finalArgs = c.MassageArgs(i.finalArgs)
	}
	return i
}

// String returns this invocation roughly as a copy-pastable shell command.
// Note: if args contain quotes some manual editing may be required.
func (i invocation) String() string {
	return fmt.Sprintf("%s %s", i.name, quotedArgsString(i.finalArgs))
}

func (c *Bin) configureCommand(i invocation) (*PreparedCmd, error) {
	executed := newExecutedCMD(i)

	cmd, cancel := c.cmd(i.finalArgs)

	outFile, errFile, combinedFile :=
		mustOpenFileAppendOnly(c.LogDir, "stdout"),
		mustOpenFileAppendOnly(c.LogDir, "stderr"),
		mustOpenFileAppendOnly(c.LogDir, "combined")

	allFiles := io.MultiWriter(outFile, errFile, combinedFile)

	stdoutWriters := []io.Writer{outFile, combinedFile, executed.Stdout, executed.Combined}
	stderrWriters := []io.Writer{errFile, combinedFile, executed.Stderr, executed.Combined}

	if !quiet() {
		stdout, err := prefixedPipe("%s:%s:stdout> ", c.TestName, c.InstanceName)
		if err != nil {
			return nil, err
		}
		stderr, err := prefixedPipe("%s:%s:stderr> ", c.TestName, c.InstanceName)
		if err != nil {
			return nil, err
		}
		stdoutWriters = append(stdoutWriters, stdout)
		stderrWriters = append(stderrWriters, stderr)
	}

	cmd.Stdout = io.MultiWriter(stdoutWriters...)
	cmd.Stderr = io.MultiWriter(stderrWriters...)

	preRun := func() error {
		relPath := "/"
		if cmd.Dir != "" {
			relPath += mustGetRelPath(c.RootDir, cmd.Dir)
		}
		cmdStr := fmt.Sprintf("%s$> %s", relPath, i)
		rtLog("%s:%s", c.ID(), cmdStr)
		fmt.Fprintf(allFiles, cmdStr+"\n")
		return nil
	}

	postRun := func() error {
		defer func() {
			cancel()
			if err := closeFiles(outFile, errFile, combinedFile); err != nil {
				panic(err)
			}
		}()
		if !c.ShouldStillBeRunningAfterTest || !cmd.ProcessState.Exited() {
			return nil
		}
		exitCode := tryGetExitCode("cmd.ProcessState.Sys()", cmd.ProcessState.Sys())
		rtLog("%s:error:early-exit> exit code %d; combined log tail follows", c.ID(), exitCode)
		prefixedOut, err := prefixedPipe("%s:combined> ", c.ID())
		if err != nil {
			return err
		}
		fmt.Fprintf(prefixedOut, executed.Combined.String())
		return fmt.Errorf("process exited early: %s: exit code %d", c.ID(), exitCode)
	}

	return &PreparedCmd{
		Cmd:        cmd,
		Cancel:     cancel,
		PreRun:     preRun,
		PostRun:    postRun,
		executed:   executed,
		invocation: i,
	}, nil
}

func (c *PreparedCmd) runWithTimeout(timeout time.Duration) error {
	defer c.PostRun()
	c.PreRun()
	errCh := make(chan error, 1)
	go func() {
		select {
		case errCh <- c.Cmd.Run():
		case <-time.After(timeout):
			errCh <- fmt.Errorf("command timed out after %s: %s", timeout, c)
			c.Cancel()
		}
	}()
	return <-errCh
}

func mustGetRelPath(base, target string) string {
	relPath, err := filepath.Rel(base, target)
	if err != nil {
		panic(fmt.Errorf("getting relative dir: %s", err))
	}
	return relPath
}

// MustRun fails the test if the command fails; else returns the stdout from the command.
func (c *Bin) MustRun(t *testing.T, subcmd string, f Flags, args ...string) string {
	t.Helper()
	executed, err := c.Run(t, subcmd, f, args...)
	if err != nil {
		t.Fatalf("Command failed: %s; error: %s; output:\n%s", executed, err, executed.Combined)
	}
	return executed.Stdout.String()
}

// MustFail fails the test if the command succeeds with a non-zero exit code.
// If the command fails for a different reason (e.g. failure to connect pipes),
// then the test also fails, as that is not the kind of failure we are looking
// for. It returns stderr from the command.
func (c *Bin) MustFail(t *testing.T, subcmd string, f Flags, args ...string) string {
	t.Helper()
	executed, err := c.Run(t, subcmd, f, args...)
	if err == nil {
		t.Fatalf("Command should have failed: %s", executed.invocation)
	}
	_, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("Want non-zero exit code (exec.ExecError); was a %T: %s", err, err)
	}
	return executed.Stderr.String()
}
