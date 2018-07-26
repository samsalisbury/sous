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
	TestName     string
	InstanceName string
	BinName      string
	BaseDir      string
	BinPath      string
	ConfigDir    string
	LogDir       string
	// Dir is the working directory.
	Dir string
	// Env are persistent env vars to pass to invocations.
	Env map[string]string
	// MassageArgs is called on the total set of args passed to the command,
	// prior to execution; the args it returns are what is finally used.
	MassageArgs  func(*testing.T, []string) []string
	TestFinished <-chan struct{}
}

// NewBin returns a new minimal Bin, all files will be created in subdirectories
// of baseDir.
func NewBin(t *testing.T, path, name, baseDir string, finished <-chan struct{}) Bin {
	illegalChars := ":/>"
	if strings.ContainsAny(name, illegalChars) {
		log.Panicf("name %q contains at least one illegal character from %q", name, illegalChars)
	}
	binName := filepath.Base(path)
	return Bin{
		TestName:     t.Name(),
		BinPath:      path,
		InstanceName: name,
		BinName:      binName,
		BaseDir:      baseDir,
		Env:          map[string]string{},
		ConfigDir:    filepath.Join(baseDir, "config"),
		LogDir:       filepath.Join(baseDir, "logs"),
		TestFinished: finished,
	}
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
	PreRun   func()
	PostRun  func()
	executed *ExecutedCMD
}

// Run runs the command.
func (c *Bin) Run(t *testing.T, subcmd string, f Flags, args ...string) (*ExecutedCMD, error) {
	cmd := c.Command(t, subcmd, f, args...)
	err := cmd.runWithTimeout(3 * time.Minute)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return cmd.executed, err
}

// Command returns the prepared command.
func (c *Bin) Command(t *testing.T, subcmd string, f Flags, args ...string) *PreparedCmd {
	i := c.newInvocation(t, subcmd, f, args...)
	return c.configureCommand(t, i)
}

// invocation is the invocation directly from the test, without any formatting
// or manipulation.
type invocation struct {
	name, subcmd string
	flags        Flags
	args         []string
	finalArgs    []string
}

func (c *Bin) newInvocation(t *testing.T, subcmd string, f Flags, args ...string) invocation {
	t.Helper()
	i := invocation{name: c.BinName, subcmd: subcmd, flags: f, args: args}
	i.finalArgs = i.allArgs()
	if c.MassageArgs != nil {
		i.finalArgs = c.MassageArgs(t, i.finalArgs)
	}
	return i
}

// String returns this invocation roughly as a copy-pastable shell command.
// Note: if args contain quotes some manual editing may be required.
func (i invocation) String() string {
	return fmt.Sprintf("%s %s", i.name, strings.Join(i.finalArgs, " "))
}

func (c *Bin) configureCommand(t *testing.T, i invocation) *PreparedCmd {
	t.Helper()

	executed := newExecutedCMD(i)

	cmd, cancel := c.cmd(i.finalArgs)

	outFile, errFile, combinedFile :=
		openFileAppendOnly(t, c.LogDir, "stdout"),
		openFileAppendOnly(t, c.LogDir, "stderr"),
		openFileAppendOnly(t, c.LogDir, "combined")

	allFiles := io.MultiWriter(outFile, errFile, combinedFile)

	stdoutWriters := []io.Writer{outFile, combinedFile, executed.Stdout, executed.Combined}
	stderrWriters := []io.Writer{errFile, combinedFile, executed.Stderr, executed.Combined}

	if !quiet() {
		stdout, stderr := prefixWithTestName(t, c.InstanceName)
		stdoutWriters = append(stdoutWriters, stdout)
		stderrWriters = append(stderrWriters, stderr)
	}

	cmd.Stdout = io.MultiWriter(stdoutWriters...)
	cmd.Stderr = io.MultiWriter(stderrWriters...)

	preRun := func() {
		rtLog("%s:$> %s", c.ID(), i)
		var relPath string
		if cmd.Dir != "" {
			relPath = " " + mustGetRelPath(t, c.BaseDir, cmd.Dir)
		}
		fmt.Fprintf(allFiles, "%s> %s", relPath, i)
	}
	postRun := func() {
		if !cmd.ProcessState.Success() {
			exitCode := tryGetExitCode("cmd.ProcessState.Sys()", cmd.ProcessState.Sys())
			if exitCode != -1 {
				rtLog("%s:error> exit code %d; combined logs follow:", c.ID(), exitCode)
			}
			prefixedOut, err := prefixedPipe("%s:combined> ", c.ID())
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(prefixedOut, executed.Combined.String())
		}
		cancel()
		closeFiles(t, outFile, errFile, combinedFile)
	}

	return &PreparedCmd{
		Cmd:        cmd,
		Cancel:     cancel,
		PreRun:     preRun,
		PostRun:    postRun,
		executed:   executed,
		invocation: i,
	}
}

func (c *PreparedCmd) start() error {
	defer c.PostRun()
	c.PreRun()
	return c.Cmd.Start()
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

func mustGetRelPath(t *testing.T, base, target string) string {
	t.Helper()
	relPath, err := filepath.Rel(base, target)
	if err != nil {
		t.Fatalf("getting relative dir: %s", err)
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
// for.
func (c *Bin) MustFail(t *testing.T, subcmd string, f Flags, args ...string) {
	t.Helper()
	_, err := c.Run(t, subcmd, f, args...)
	if err == nil {
		t.Fatalf("command should have failed: sous %s", args)
	}
	_, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("want non-zero exit code (exec.ExecError); was a %T: %s", err, err)
	}
}
