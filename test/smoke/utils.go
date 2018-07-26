package smoke

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
)

// timeGoTestInvoked is used to group test data for tests run
// via the same go test invocation.
var timeGoTestInvoked = time.Now().Format(time.RFC3339)

func quiet() bool {
	return os.Getenv("SMOKE_TEST_QUIET") == "YES"
}

func addGitEnvVars(env map[string]string) {
	env["GIT_CONFIG_NOSYSTEM"] = "yes"
	env["HOME"] = "nowhere"
	env["PREFIX"] = "nowhere"
	env["GIT_COMMITTER_NAME"] = "Tester"
	env["GIT_COMMITTER_EMAIL"] = "tester@example.com"
	env["GIT_AUTHOR_NAME"] = "Tester"
	env["GIT_AUTHOR_EMAIL"] = "tester@example.com"
}

func rtLog(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

func getEnvDesc() desc.EnvDesc {
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
		log.Panicf("setup failed to decode %q: %s", descPath, err)
	}
	return envDesc
}

func mustGetSousBin() string {
	sousBin := os.Getenv("SMOKE_TEST_BINARY")
	if sousBin != "" {
		log.Printf("Using sous binary %q (from $SMOKE_TEST_BINARY)", sousBin)
		return sousBin
	}
	sousBin, err := exec.LookPath("sous")
	if err != nil {
		log.Panicf("sous not found in path and $SMOKE_TEST_BINARY not set")
	}
	log.Printf("Using sous binary %q (from $PATH)", sousBin)
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

	log.Printf("Test data in %q (from %s)", baseDir, from)
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

var lastPort int
var freePortsMu sync.Mutex
var usedPorts = map[int]struct{}{}

// freePortAddrs returns n listenable addresses on the ip provided in the
// range min-max. Note that it does not guarantee they are still free by the
// time you come to bind to them, but makes that more likely by binding and then
// unbinding from them.
func freePortAddrs(ip string, n, min, max int) []string {
	freePortsMu.Lock()
	defer freePortsMu.Unlock()
	ports := make(map[int]net.Listener, n)
	addrs := make([]string, n)
	if lastPort < min || lastPort > max {
		lastPort = min
	}
	for i := 0; i < n; i++ {
		p, addr, listener, err := oneFreePort(ip, lastPort, min, max)
		if err != nil {
			log.Panic(err)
		}
		lastPort = p
		addrs[i] = addr
		ports[p] = listener
		usedPorts[p] = struct{}{}
	}
	// Now release them all. It's now a race to get our desired things
	// listening on these addresses.
	for _, l := range ports {
		if err := l.Close(); err != nil {
			log.Panic(err)
		}
	}
	return addrs
}

func oneFreePort(ip string, start, min, max int) (int, string, net.Listener, error) {
	port := start
	maxAttempts := max - min
	for a := 0; a < maxAttempts; a, port = a+1, port+1 {
		if port > max {
			port = min
		}
		if _, ok := usedPorts[port]; ok {
			continue
		}
		addr := fmt.Sprintf("%s:%d", ip, port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			if listener != nil {
				if err := listener.Close(); err != nil {
					return 0, "", nil, fmt.Errorf("failed to close listener: %s", err)
				}
			}
			continue
		}
		return port, addr, listener, nil
	}
	return 0, "", nil, fmt.Errorf("unable to find a free port in range %d-%d", min, max)
}

func prefixedPipe(prefix string) (io.Writer, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	go func() {
		defer func() {
			if err := r.Close(); err != nil {
				rtLog("Failed to close reader: %s", err)
			}
			if err := w.Close(); err != nil {
				rtLog("Failed to close writer: %s", err)
			}
		}()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				rtLog("Error prefixing: %s", err)
			}
			fmt.Fprintf(os.Stdout, "%s%s\n", prefix, scanner.Text())
		}
	}()
	return w, nil
}

func prefixWithTestName(t *testing.T, label string) (prefixedOut, prefixedErr io.Writer) {
	t.Helper()

	outPrefix := fmt.Sprintf("%s:%s:stdout> ", t.Name(), label)
	errPrefix := fmt.Sprintf("%s:%s:stderr> ", t.Name(), label)

	stdout, err := prefixedPipe(outPrefix)
	if err != nil {
		t.Fatalf("Setting up output prefix: %s", err)
	}
	stderr, err := prefixedPipe(errPrefix)
	if err != nil {
		t.Fatalf("Setting up output prefix: %s", err)
	}
	return stdout, stderr
}
