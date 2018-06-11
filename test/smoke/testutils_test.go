//+build smoke

package smoke

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
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
	t.Helper()
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

func openFileAppendOnly(t *testing.T, baseDir, fileName string) *os.File {
	t.Helper()
	time.Sleep(time.Second)
	filePath := path.Join(baseDir, fileName)
	assertDirNotExists(t, filePath)
	if !fileExists(t, filePath) {
		makeFile(t, baseDir, fileName, nil)
	}
	file, err := os.OpenFile(filePath,
		os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0x777)
	if err != nil {
		t.Fatalf("opening file for append: %s", err)
	}
	return file
}

// makeFileString is a convenience wrapper around makeFile, using string s
// as the bytes to be written.
func makeFileString(t *testing.T, baseDir, fileName string, s string) string {
	t.Helper()
	return makeFile(t, baseDir, fileName, []byte(s))
}

func fileExists(t *testing.T, filePath string) bool {
	t.Helper()
	s, err := os.Stat(filePath)
	if err == nil {
		return s.Mode().IsRegular()
	}
	if isNotExist(err) {
		return false
	}
	t.Fatalf("checking if file exists: %s", err)
	return false
}

func assertDirNotExists(t *testing.T, filePath string) {
	t.Helper()
	if dirExists(t, filePath) {
		t.Fatalf("%s exists and is a directory", filePath)
	}
}

func dirExists(t *testing.T, filePath string) bool {
	t.Helper()
	s, err := os.Stat(filePath)
	if err == nil {
		return s.IsDir()
	}
	if isNotExist(err) {
		return false
	}
	t.Fatalf("checking if dir exists: %s", err)
	return false
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

func getDataDir(t *testing.T) string {
	baseDir := os.Getenv("SMOKE_TEST_DATA_DIR")
	from := "$SMOKE_TEST_DATA_DIR"
	if baseDir == "" {
		baseDir = path.Join(os.TempDir(), timeGoTestInvoked)
		from = "$TMPDIR"
	}

	baseDir = path.Join(baseDir, t.Name())

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
	c, cancel := mkCMD(dir, name, args...)
	defer cancel()
	if o, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("command %q %v in dir %q failed: %v; output was:\n%s",
			name, args, dir, err, string(o))
	}
	return nil
}

var perCommandTimeout = 5 * time.Minute

func mkCMD(dir, name string, args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), perCommandTimeout)
	c := exec.CommandContext(ctx, name, args...)
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
	c.Dir = dir
	return c, cancel
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

func closeFile(t *testing.T, f *os.File) (ok bool) {
	t.Helper()
	if err := f.Close(); err != nil {
		t.Errorf("failed to close %s: %s", f.Name(), err)
		return false
	}
	return true
}

func closeFiles(t *testing.T, fs ...*os.File) {
	t.Helper()
	var closeFailed bool
	for _, f := range fs {
		if !closeFile(t, f) {
			closeFailed = true
		}
	}
	if closeFailed {
		t.Fatalf("failed to close some files, see above")
	}
}

// freePortAddrs returns n listenable addresses on the ip provided in the
// range min-max. Note that it does not guarantee they are still free by the
// time you come to bind to them, but makes that more likely by binding and then
// unbinding from them.
func freePortAddrs(t *testing.T, ip string, n, min, max int) []string {
	t.Helper()
	ports := make(map[int]net.Listener, n)
	addrs := make([]string, n)
	// First bind to all the ports...
	port := min
NEXT_PORT:
	for i := 0; i < n; i++ {
		if port > max {
			port = min
		}
		for ; port <= max; port++ {
			addr := fmt.Sprintf("%s:%d", ip, port)
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				port = port + 1
				continue
			}
			addrs[i] = addr
			ports[port] = listener
			continue NEXT_PORT
		}
		t.Fatalf("Unable to find a free port.")
	}
	// Now release them all. It's now a race to get our desired things
	// listening on these addresses.
	for _, l := range ports {
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return addrs
}

func prefixWithTestName(t *testing.T, label string) (prefixedOut, prefixedErr io.Writer) {
	t.Helper()
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Setting up output prefix: %s", err)
	}
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Setting up output prefix: %s", err)
	}
	go func() {
		defer func() {
			if err := outReader.Close(); err != nil {
				t.Fatalf("Failed to close outReader: %s", err)
			}
			if err := outWriter.Close(); err != nil {
				t.Fatalf("Failed to close outWriter: %s", err)
			}
		}()
		scanner := bufio.NewScanner(outReader)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				t.Fatalf("Error prefixing stdout: %s", err)
			}
			fmt.Fprintf(os.Stdout, "%s:%s:stdout > %s\n", t.Name(), label, scanner.Text())
		}
	}()
	go func() {
		defer func() {
			if err := errReader.Close(); err != nil {
				t.Fatalf("Failed to close errReader: %s", err)
			}
			if err := errWriter.Close(); err != nil {
				t.Fatalf("Failed to close errWriter: %s", err)
			}
		}()
		scanner := bufio.NewScanner(errReader)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				t.Fatalf("Error prefixing stderr: %s", err)
			}
			fmt.Fprintf(os.Stderr, "%s:%s:stderr > %s\n", t.Name(), label, scanner.Text())
		}
	}()
	return outWriter, errWriter
}
