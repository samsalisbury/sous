//+build smoke

package smoke

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/docker"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/yaml"
)

type TestClient struct {
	BaseDir   string
	BinPath   string
	ConfigDir string
	LogDir    string
	// Dir is the working directory.
	Dir           string
	ClusterSuffix string
}

type sousFlags struct {
	kind    string
	flavor  string
	cluster string
	repo    string
	offset  string
	tag     string
}

func (f *sousFlags) Args() []string {
	if f == nil {
		return nil
	}
	var out []string
	if f.kind != "" {
		out = append(out, "-kind", f.kind)
	}
	if f.flavor != "" {
		out = append(out, "-flavor", f.flavor)
	}
	if f.cluster != "" {
		out = append(out, "-cluster", f.cluster)
	}
	if f.repo != "" {
		out = append(out, "-repo", f.repo)
	}
	if f.offset != "" {
		out = append(out, "-offset", f.offset)
	}
	if f.tag != "" {
		out = append(out, "-tag", f.tag)
	}
	return out
}

func makeClient(baseDir, sousBin string) *TestClient {
	baseDir = path.Join(baseDir, "client1")
	return &TestClient{
		BaseDir:   baseDir,
		BinPath:   sousBin,
		ConfigDir: path.Join(baseDir, "config"),
		LogDir:    path.Join(baseDir, "logs"),
	}
}

func (c *TestClient) Configure(server, dockerReg, userEmail string) error {
	if err := os.MkdirAll(c.ConfigDir, 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(c.LogDir, 0777); err != nil {
		return err
	}
	user := strings.Split(userEmail, "@")
	conf := config.Config{
		Server: server,
		Docker: docker.Config{
			RegistryHost: dockerReg,
		},
		User: sous.User{
			Name:  user[0],
			Email: userEmail,
		},
	}
	conf.PollIntervalForClient = 600

	clientDebug := os.Getenv("SOUS_CLIENT_DEBUG") == "true"

	if clientDebug {
		conf.Logging.Basic.Level = "ExtraDebug1"
		conf.Logging.Basic.DisableConsole = false
		conf.Logging.Basic.ExtraConsole = true
	}

	y, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(c.ConfigDir, "config.yaml"), y, os.ModePerm); err != nil {
		return err
	}
	return nil
}

// allArgs produces a []string representing all args determined by the sous
// subcommand, sous flags and any other args.
func allArgs(subcmd string, f *sousFlags, args []string) []string {
	allArgs := strings.Split(subcmd, " ")
	allArgs = append(allArgs, f.Args()...)
	allArgs = append(allArgs, args...)
	return allArgs
}

func insertClusterSuffix(args []string, suffix string) []string {
	for i, s := range args {
		if s == "-cluster" && len(args) > i+1 {
			args[i+1] = args[i+1] + suffix
		}
		if s == "-tag" && len(args) > i+1 {
			args[i+1] = args[i+1] + "-" + strings.Replace(suffix, "_", "-", -1)
		}
	}
	return args
}

func (c *TestClient) Cmd(t *testing.T, subcmd string, f *sousFlags, args ...string) (*exec.Cmd, context.CancelFunc) {
	t.Helper()
	args = insertClusterSuffix(args, c.ClusterSuffix)
	cmd, cancel := mkCMD(c.Dir, c.BinPath, allArgs(subcmd, f, args)...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("SOUS_CONFIG_DIR=%s", c.ConfigDir))
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

func (c *TestClient) Run(t *testing.T, subcmd string, f *sousFlags, args ...string) (*ExecutedCMD, error) {
	t.Helper()
	cmd, cancel := c.Cmd(t, subcmd, f, args...)
	defer cancel()
	stdout, stderr := prefixWithTestName(t, "client1")
	fmt.Fprintf(stderr, "SOUS_CONFIG_DIR = %q\n", c.ConfigDir)
	fmt.Fprintf(stdout, "running sous in %q: %s\n", c.Dir, args)
	args = quotedArgs(args)
	outFile, errFile, combinedFile :=
		openFileAppendOnly(t, c.LogDir, "stdout"),
		openFileAppendOnly(t, c.LogDir, "stderr"),
		openFileAppendOnly(t, c.LogDir, "combined")

	defer closeFiles(t, outFile, errFile, combinedFile)

	allFiles := io.MultiWriter(outFile, errFile, combinedFile)

	executed := newExecutedCMD(subcmd, args)

	cmd.Stdout = io.MultiWriter(stdout, outFile, combinedFile, executed.Stdout, executed.Combined)
	cmd.Stderr = io.MultiWriter(stderr, errFile, combinedFile, executed.Stderr, executed.Combined)
	prettyCmd := fmt.Sprintf("$ sous %s\n", strings.Join(allArgs(subcmd, f, args), " "))
	fmt.Fprintf(os.Stderr, "==> %s", prettyCmd)
	relPath := mustGetRelPath(t, c.BaseDir, cmd.Dir)
	fmt.Fprintf(allFiles, "%s %s", relPath, prettyCmd)
	err := runWithTimeout(cmd, cancel, 3*time.Minute)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return executed, err
}

func runWithTimeout(cmd *exec.Cmd, cancel context.CancelFunc, timeout time.Duration) error {
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Run()
	}()
	go func() {
		<-time.After(timeout)
		errCh <- fmt.Errorf("command timed out after %s:\nsous %s", timeout,
			quotedArgsString(cmd.Args[1:]))
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
func (c *TestClient) MustRun(t *testing.T, subcmd string, f *sousFlags, args ...string) string {
	t.Helper()
	executed, err := c.Run(t, subcmd, f, args...)
	if err != nil {
		t.Logf("Command failed: %s; output:\n%s", executed, executed.Combined)
		t.Fatal(err)
	}
	return executed.Stdout.String()
}

func (c *TestClient) MustFail(t *testing.T, subcmd string, f *sousFlags, args ...string) {
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

func (c *TestClient) TransformManifestAsString(t *testing.T, getSetFlags *sousFlags, f func(manifest string) string) {
	manifest := c.MustRun(t, "manifest get", getSetFlags)
	manifest = f(manifest)
	manifestSetCmd, cancel := c.Cmd(t, "manifest set", getSetFlags)
	defer cancel()
	manifestSetCmd.Stdin = ioutil.NopCloser(bytes.NewReader([]byte(manifest)))
	if out, err := manifestSetCmd.CombinedOutput(); err != nil {
		t.Fatalf("manifest set failed: %s; output:\n%s", err, out)
	}
}

func (c *TestClient) TransformManifest(t *testing.T, getSetFlags *sousFlags, f func(m sous.Manifest) sous.Manifest) {
	t.Helper()
	manifest := c.MustRun(t, "manifest get", getSetFlags)
	var m sous.Manifest
	if err := yaml.Unmarshal([]byte(manifest), &m); err != nil {
		t.Fatalf("manifest get returned invalid YAML: %s\nInvalid YAML was:\n%s", err, manifest)
	}
	m = f(m)
	manifestBytes, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal updated manifest: %s\nInvalid manifest was:\n% #v", err, m)
	}
	manifestSetCmd, cancel := c.Cmd(t, "manifest set", getSetFlags)
	defer cancel()
	manifestSetCmd.Stdin = ioutil.NopCloser(bytes.NewReader(manifestBytes))
	if out, err := manifestSetCmd.CombinedOutput(); err != nil {
		t.Fatalf("manifest set failed: %s; output:\n%s", err, out)
	}
}
