package testagents

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/prefixpipe"
)

var perCommandTimeout = 5 * time.Minute

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
	MassageArgs func([]string) []string
	Finished    chan struct{}

	// ShouldStillBeRunningAfterTest should is set to true for servers etc, it
	// enables crash detection.
	ShouldStillBeRunningAfterTest bool

	// Verbose if set to true, print all stdout and stderr output inline.
	// Note this output is printed to log files regardless.
	Verbose bool

	// LogFunc is called with realtime logs of command invocations and output
	// etc. Defaults to log.Printf from stdlib.
	LogFunc func(string, ...interface{})

	// ProcMan keeps track of PIDs created so you can clean them up later.
	ProcMan ProcMan

	longLivedPipes []*prefixpipe.PrefixPipe
}

// NewBin returns a new minimal Bin, all files will be created in subdirectories
// of baseDir.
func NewBin(t *testing.T, pm ProcMan, path, name, baseDir, rootDir string, finished chan struct{}) Bin {
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
		Finished:     finished,
		LogFunc:      log.Printf,
		ProcMan:      pm,
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

func mkCMD(dir, name string, args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), perCommandTimeout)
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = dir
	return c, cancel
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

// Run runs the command.
func (c *Bin) Run(t *testing.T, subcmd string, f Flags, args ...string) (*ExecutedCMD, error) {
	cmd, err := c.Command(nil, subcmd, f, args...)
	if err != nil {
		return nil, fmt.Errorf("setting up command failed: %s", err)
	}
	err = cmd.runWithTimeout(3 * time.Minute)
	if err != nil {
		log.Printf("Command failed: %s: %s", cmd.invocation, err)
	}
	c.waitPipes()
	return cmd.executed, err
}

func (c *Bin) waitPipes() error {
	var errs []error
	for _, p := range c.longLivedPipes {
		if err := p.Wait(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("multiple errors: % #v", errs)
}

// RunWithStdin runs the command, attaching stdin.
func (c *Bin) RunWithStdin(t *testing.T, stdin io.ReadCloser, subcmd string, f Flags, args ...string) (*ExecutedCMD, error) {
	cmd, err := c.Command(stdin, subcmd, f, args...)
	if err != nil {
		return nil, fmt.Errorf("setting up command failed: %s", err)
	}
	err = cmd.runWithTimeout(3 * time.Minute)
	if err != nil {
		log.Printf("Command failed: %s: %s", cmd.invocation, err)
	}
	return cmd.executed, err
}

// Command returns the prepared command.
func (c *Bin) Command(stdin io.ReadCloser, subcmd string, f Flags, args ...string) (*PreparedCmd, error) {
	i := c.newInvocation(subcmd, f, args...)
	return c.configureCommand(i, stdin)
}

func (c *Bin) newInvocation(subcmd string, f Flags, args ...string) invocation {
	i := invocation{name: c.BinName, subcmd: subcmd, flags: f, args: args}
	i.finalArgs = i.allArgs()
	if c.MassageArgs != nil {
		i.finalArgs = c.MassageArgs(i.finalArgs)
	}
	return i
}

func (c *Bin) printStdout() bool {
	return c.Verbose
}

func (c *Bin) printStderr() bool {
	return c.Verbose
}

func (c *Bin) prefix(label string) string {
	return fmt.Sprintf("%s:%s> ", c.ID(), label)
}

func (c *Bin) prefixWriter(label string) *prefixpipe.PrefixPipe {
	w := prefixpipe.New(os.Stdout, c.prefix(label))
	c.longLivedPipes = append(c.longLivedPipes, w)
	return w
}

func (c *Bin) prefixPrintf(label, format string, a ...interface{}) {
	w := prefixpipe.New(os.Stdout, c.prefix(label))
	defer func() {
		if err := w.Close(); err != nil {
			log.Panicf("unable to close prefix pipe: %s", err)
		}
	}()
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		log.Panicf("unable to write prefixed log string")
	}
}

func (c *Bin) configureCommand(i invocation, stdin io.ReadCloser) (*PreparedCmd, error) {
	executed := newExecutedCMD(i)

	cmd, cancel := c.cmd(i.finalArgs)

	cmd.Stdin = stdin

	outFile, errFile, combinedFile :=
		mustOpenFileAppendOnly(c.LogDir, "stdout"),
		mustOpenFileAppendOnly(c.LogDir, "stderr"),
		mustOpenFileAppendOnly(c.LogDir, "combined")

	allFiles := io.MultiWriter(outFile, errFile, combinedFile)

	stdoutWriters := []io.Writer{outFile, combinedFile, executed.Stdout, executed.Combined}
	stderrWriters := []io.Writer{errFile, combinedFile, executed.Stderr, executed.Combined}

	if c.printStdout() {
		stdoutWriters = append(stdoutWriters, c.prefixWriter("stdout"))
	}
	if c.printStderr() {
		stderrWriters = append(stderrWriters, c.prefixWriter("stderr"))
	}

	cmd.Stdout = io.MultiWriter(stdoutWriters...)
	cmd.Stderr = io.MultiWriter(stderrWriters...)

	preRun := func() error {
		relPath := "/"
		if cmd.Dir != "" {
			relPath += mustGetRelPath(c.RootDir, cmd.Dir)
		}
		cmdStr := fmt.Sprintf("%s$> %s", relPath, i)
		c.LogFunc("%s:%s", c.ID(), cmdStr)
		fmt.Fprintf(allFiles, cmdStr+"\n")
		return nil
	}

	var pc *PreparedCmd

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
		exitCode, err := pc.tryGetExitCode()
		// TODO SS: print just tail of logs, with configurable nu lines.
		suffix := "; combined log output follows"
		if err != nil {
			c.prefixPrintf("error:early-exit", "unable to determine exit code: %s"+suffix, err)
		} else {
			c.prefixPrintf("error:early-exit", "exit code: %s"+suffix, exitCode)
		}
		c.prefixPrintf("logs/combined", executed.Combined.String())
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

func fileExists(filePath string) bool {
	s, err := os.Stat(filePath)
	if err == nil {
		return s.Mode().IsRegular()
	}
	if isNotExist(err) {
		return false
	}
	panic(fmt.Errorf("checking if file exists: %s", err))
}

func isNotExist(err error) bool {
	if err == nil {
		panic("cannot check nil error")
	}
	return err == os.ErrNotExist ||
		strings.Contains(err.Error(), "no such file or directory")
}

func mustOpenFileAppendOnly(baseDir, fileName string) *os.File {
	filePath := path.Join(baseDir, fileName)
	assertDirNotExists(filePath)
	if !fileExists(filePath) {
		makeFile(baseDir, fileName, nil)
	}
	file, err := os.OpenFile(filePath,
		os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0x777)
	if err != nil {
		panic(fmt.Errorf("opening file for append: %s", err))
	}
	return file
}

func closeFiles(fs ...*os.File) error {
	var failures []string
	for _, f := range fs {
		if err := f.Close(); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", f.Name(), err))
		}
	}
	if len(failures) == 0 {
		return nil
	}
	return fmt.Errorf("failed to close files: %s", strings.Join(failures, "; "))
}

func assertDirNotExists(filePath string) {
	if dirExists(filePath) {
		panic(fmt.Errorf("%s exists and is a directory", filePath))
	}
}

// makeFile attempts to write bytes to baseDir/fileName and returns the full
// path to the file. It assumes the directory baseDir already exists and
// contains no file named fileName, and will fail otherwise.
func makeFile(baseDir, fileName string, bytes []byte) string {
	filePath := path.Join(baseDir, fileName)
	if _, err := os.Open(filePath); err != nil {
		if !isNotExist(err) {
			panic(fmt.Errorf("unable to check if file %q exists: %s", filePath, err))
		}
	} else {
		panic(fmt.Errorf("file %q already exists", filePath))
	}

	if err := ioutil.WriteFile(filePath, bytes, 0777); err != nil {
		panic(fmt.Errorf("unable to write file %q: %s", filePath, err))
	}
	return filePath
}

// makeFileString is a convenience wrapper around makeFile, using string s
// as the bytes to be written.
func makeFileString(baseDir, fileName string, s string) string {
	return makeFile(baseDir, fileName, []byte(s))
}

func dirExists(filePath string) bool {
	s, err := os.Stat(filePath)
	if err == nil {
		return s.IsDir()
	}
	if isNotExist(err) {
		return false
	}
	panic(fmt.Errorf("checking if dir exists: %s", err))
}
