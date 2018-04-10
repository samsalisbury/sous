//+build smoke

package smoke

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
)

// timeGoTestInvoked is used to group test data for tests run
// via the same go test invocation.
var timeGoTestInvoked = time.Now().Format(time.RFC3339)

func getEnvDesc(t *testing.T) desc.EnvDesc {
	descPath := os.Getenv("SOUS_QA_DESC")
	if descPath == "" {
		panic("SOUS_QA_DESC is unset! See sous_qa_setup.")
	}
	descReader, err := os.Open(descPath)
	if err != nil {
		panic(err)
	}
	var envDesc desc.EnvDesc
	if err := json.NewDecoder(descReader).Decode(&envDesc); err != nil {
		t.Fatalf("setup failed to decode %q: %s", descPath, err)
	}
	return envDesc
}

func getSousBin(t *testing.T) string {
	sousBin := os.Getenv("SMOKE_TEST_BINARY")
	if sousBin != "" {
		t.Logf("Using sous binary %q (from $SMOKE_TEST_BINARY)", sousBin)
		return sousBin
	}
	sousBin, err := exec.LookPath("sous")
	if err != nil {
		t.Fatalf("sous not found in path")
	}
	t.Logf("Using sous binary %q (from $PATH)", sousBin)
	return sousBin
}

// makeEmptyDir safely creates an empty dir "dir" inside baseDir and returns the
// full path.
func makeEmptyDir(t *testing.T, baseDir, dir string) string {
	t.Helper()
	dir = path.Join(baseDir, dir)
	if dirExistsAndIsNotEmpty(t, dir) {
		t.Fatalf("dir %q already exists and is not empty", dir)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatal(err)
	}
	return dir
}

// makeFile attempts to write bytes to baseDir/fileName and returns the full
// path to the file. It assumes the directory baseDir already exists and
// contains no file named fileName, and will fail otherwise.
func makeFile(t *testing.T, baseDir, fileName string, bytes []byte) string {
	filePath := path.Join(baseDir, fileName)
	if _, err := os.Open(filePath); err != nil {
		if !isNotExist(err) {
			t.Fatalf("unable to check if file %q exists: %s", filePath, err)
		}
	} else {
		t.Fatalf("file %q already exists", filePath)
	}

	if err := ioutil.WriteFile(filePath, bytes, 0777); err != nil {
		t.Fatalf("unable to write file %q: %s", filePath, err)
	}
	return filePath
}

// makeFileString is a convenience wrapper around makeFile, using string s
// as the bytes to be written.
func makeFileString(t *testing.T, baseDir, fileName string, s string) string {
	t.Helper()
	return makeFile(t, baseDir, fileName, []byte(s))
}

func dirExistsAndIsNotEmpty(t *testing.T, baseDir string) bool {
	t.Helper()
	f, err := os.Open(baseDir)
	if err != nil {
		if isNotExist(err) {
			return false
		}
		t.Fatalf("Could not check dir not exists or empty: %s", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("failed to close file handle: %s", err)
		}
	}()
	_, err = f.Readdirnames(1)
	return err == nil || (err != io.EOF)
}

func getDataDir(t *testing.T, testName string) string {
	baseDir := os.Getenv("SMOKE_TEST_DATA_DIR")
	from := "$SMOKE_TEST_DATA_DIR"
	if baseDir == "" {
		baseDir = path.Join(os.TempDir(), timeGoTestInvoked)
		from = "$TMPDIR"
	}

	baseDir = path.Join(baseDir, testName)

	// Check dir does not exist or is at least empty.
	if dirExistsAndIsNotEmpty(t, baseDir) {
		t.Fatalf("Test data dir already exists and is not empty: %q", baseDir)
	}

	t.Logf("Writing test data to %q (from %s)", baseDir, from)
	if err := os.MkdirAll(baseDir, 0777); err != nil {
		t.Fatalf("Failed to create smoke test data dir %q: %s", baseDir, err)
	}
	return baseDir
}

// addURLsToState pokes URLs from the env desc into the state.
func addURLsToState(state *sous.State, envDesc desc.EnvDesc) {
	for _, c := range state.Defs.Clusters {
		c.BaseURL = envDesc.SingularityURL()
	}
	state.Defs.DockerRepo = envDesc.RegistryName()
}

func mustDoCMD(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	if err := doCMD(dir, name, args...); err != nil {
		t.Fatal(err)
	}
}

func doCMD(dir, name string, args ...string) error {
	c := mkCMD(dir, name, args...)
	if o, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("command %q %v in dir %q failed: %v; output was:\n%s",
			name, args, dir, err, string(o))
	}
	return nil
}

func mkCMD(dir, name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Env = os.Environ()
	// Isolate git...
	c.Env = append(c.Env,
		"GIT_CONFIG_NOSYSTEM=yes",
		"HOME=/nowhere",
		"PREFIX=/nowhere",
		"GIT_COMMITTER_NAME=Tester",
		"GIT_COMMITTER_EMAIL=tester@example.com",
		"GIT_AUTHOR_NAME=Tester",
		"GIT_AUTHOR_EMAIL=tester@example.com",
	)
	return c
}

// testing os.ErrNotExist seems to not work in the majority of cases,
// at least on Darwin.
// TODO SS: Find out why...
func isNotExist(err error) bool {
	if err == nil {
		panic("cannot check nil error")
	}
	return err == os.ErrNotExist ||
		strings.Contains(err.Error(), "no such file or directory")
}

func closeFile(t *testing.T, f *os.File) {
	if err := f.Close(); err != nil {
		t.Errorf("failed to close %s: %s", pidFile, err)
	}
}
