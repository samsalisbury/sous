package smoke

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	BaseDir   string
	BinPath   string
	ConfigDir string
	LogDir    string
	// Dir is the working directory.
	Dir string
	// Env are persistent env vars to pass to invocations.
	Env map[string]string
	// MassageArgs is called on the total set of args passed to the command,
	// prior to execution; the args it returns are what is finally used.
	MassageArgs func(*testing.T, []string) []string
}

// Configure writes fm files relative to c.ConfigPath and ensures the log
// directory exists.
func (c *Bin) Configure(fm filemap.FileMap) error {
	if err := os.MkdirAll(c.ConfigDir, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(c.LogDir, os.ModePerm); err != nil {
		return err
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
func allArgs(subcmd string, f Flags, args []string) []string {
	all := strings.Split(subcmd, " ")
	all = append(all, flagArgs(f)...)
	all = append(all, args...)
	return all
}

// Cmd generates an *exec.Cmd and cancellation func.
func (c *Bin) Cmd(t *testing.T, subcmd string, f Flags, args ...string) (*exec.Cmd, context.CancelFunc) {
	t.Helper()
	args = allArgs(subcmd, f, args)
	if c.MassageArgs != nil {
		args = c.MassageArgs(t, args)
	}
	cmd, cancel := mkCMD(c.Dir, c.BinPath, args...)
	for name, value := range c.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}
	return cmd, cancel
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
	Subcmd                   string
	Args                     []string
	Stdout, Stderr, Combined *bytes.Buffer
}

// String returns something looking like a shell invocation of this command.
func (e *ExecutedCMD) String() string {
	return fmt.Sprintf("sous %s %s", e.Subcmd, quotedArgsString(e.Args))
}

func newExecutedCMD(subcmd string, args []string) *ExecutedCMD {
	return &ExecutedCMD{
		Subcmd:   subcmd,
		Args:     args,
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Combined: &bytes.Buffer{},
	}
}

type cmdWithHooks struct {
	Cmd      *exec.Cmd
	Cancel   func()
	PreRun   func()
	PostRun  func()
	executed *ExecutedCMD
}

// Run runs the command.
func (c *Bin) Run(t *testing.T, subcmd string, f Flags, args ...string) (*ExecutedCMD, error) {
	cmd := c.configureCommand(t, subcmd, f, args...)
	err := cmd.runWithTimeout(3 * time.Minute)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return cmd.executed, err
}

func (c *Bin) configureCommand(t *testing.T, subcmd string, f Flags, args ...string) *cmdWithHooks {
	t.Helper()
	cmd, cancel := c.Cmd(t, subcmd, f, args...)

	qArgs := quotedArgs(args)

	outFile, errFile, combinedFile :=
		openFileAppendOnly(t, c.LogDir, "stdout"),
		openFileAppendOnly(t, c.LogDir, "stderr"),
		openFileAppendOnly(t, c.LogDir, "combined")

	allFiles := io.MultiWriter(outFile, errFile, combinedFile)

	executed := newExecutedCMD(subcmd, qArgs)

	stdoutWriters := []io.Writer{outFile, combinedFile, executed.Stdout, executed.Combined}
	stderrWriters := []io.Writer{errFile, combinedFile, executed.Stderr, executed.Combined}

	clientName := "client1"

	if !quiet() {
		stdout, stderr := prefixWithTestName(t, clientName)
		stdoutWriters = append(stdoutWriters, stdout)
		stderrWriters = append(stderrWriters, stderr)
	}

	cmd.Stdout = io.MultiWriter(stdoutWriters...)
	cmd.Stderr = io.MultiWriter(stderrWriters...)

	preRun := func() {
		prettyCmd := fmt.Sprintf("$ sous %s", strings.Join(allArgs(subcmd, f, qArgs), " "))
		fmt.Fprintf(os.Stderr, "%s:%s:command> %s\n", t.Name(), clientName, prettyCmd)
		relPath := mustGetRelPath(t, c.BaseDir, cmd.Dir)
		fmt.Fprintf(allFiles, "%s %s", relPath, prettyCmd)
	}
	postRun := func() {
		cancel()
		closeFiles(t, outFile, errFile, combinedFile)
	}

	return &cmdWithHooks{
		Cmd:      cmd,
		Cancel:   cancel,
		PreRun:   preRun,
		PostRun:  postRun,
		executed: executed,
	}
}

func (c *cmdWithHooks) runWithTimeout(timeout time.Duration) error {
	defer c.PostRun()
	c.PreRun()
	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Cmd.Run()
	}()
	go func() {
		<-time.After(timeout)
		errCh <- fmt.Errorf("command timed out after %s:\nsous %s", timeout,
			quotedArgsString(c.Cmd.Args[1:]))
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
