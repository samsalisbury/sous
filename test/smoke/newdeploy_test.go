package smoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/go-ps"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
)

type Fixture struct {
	EnvDesc desc.EnvDesc
	Cluster TestCluster
	Client  TestClient
}

type TestCluster struct {
	BaseDir      string
	RemoteGDMDir string
	Count        int
	Instances    []*Instance
}

type Instance struct {
	Addr                string
	StateDir, ConfigDir string
	ClusterName         string
	Proc                *os.Process
	LogDir              string
}

type TestClient struct {
	BinPath   string
	ConfigDir string
	// Dir is the working directory.
	Dir string
}

func setupEnv(t *testing.T) Fixture {
	t.Helper()
	// TODO SS: make this configurable by env var
	// so can decide which bin to run against.
	sousBin, err := exec.LookPath("sous")
	if err != nil {
		t.Fatalf("sous not found in path")
	}
	t.Logf("Server and client all using sous at %s", sousBin)

	stopPIDs(t)
	if testing.Short() {
		t.Skipf("-short flag present")
	}
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

	baseDir := path.Join(os.TempDir(), "sous-test-cluster", time.Now().Format(time.RFC3339))

	state := sous.StateFixture(sous.StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
	})

	for _, c := range state.Defs.Clusters {
		c.BaseURL = envDesc.SingularityURL()
	}
	state.Defs.DockerRepo = envDesc.RegistryName()

	c, err := newTestCluster(state, baseDir)
	if err != nil {
		t.Fatalf("setting up test cluster: %s", err)
	}

	if err := c.Configure(envDesc); err != nil {
		t.Fatalf("configuring test cluster: %s", err)
	}

	if err := c.Start(t, sousBin); err != nil {
		t.Fatalf("starting test cluster: %s", err)
	}

	client := makeClient(baseDir, sousBin)
	primaryServer := "http://" + c.Instances[0].Addr
	if err := client.Configure(primaryServer, envDesc.RegistryName()); err != nil {
		t.Fatal(err)
	}

	return Fixture{
		Cluster: *c,
		Client:  client,
	}
}

func makeClient(baseDir, sousBin string) TestClient {
	baseDir = path.Join(baseDir, "client1")
	return TestClient{
		BinPath:   sousBin,
		ConfigDir: path.Join(baseDir, "config"),
	}
}

func (c *TestClient) Configure(server, dockerReg string) error {
	if err := os.MkdirAll(c.ConfigDir, 0777); err != nil {
		return err
	}
	conf := config.Config{
		Server: server,
		Docker: docker.Config{
			RegistryHost: dockerReg,
		},
		User: sous.User{
			Name:  "Sous Client1",
			Email: "sous-client1@example.com",
		},
	}
	conf.PollIntervalForClient = 600
	conf.Logging.Basic.Level = "ExtraDebug1"
	conf.Logging.Basic.DisableConsole = false
	conf.Logging.Basic.ExtraConsole = true
	y, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(c.ConfigDir, "config.yaml"), y, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func (f *Fixture) Stop(t *testing.T) {
	t.Helper()
	f.Cluster.Stop(t)
}

func (c *TestCluster) Stop(t *testing.T) {
	t.Helper()
	stopPIDs(t)
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
	return c
}

func newTestCluster(state *sous.State, baseDir string) (*TestCluster, error) {
	if err := os.MkdirAll(baseDir, 0777); err != nil {
		return nil, err
	}

	gdmDir := path.Join(baseDir, "remote-gdm-temp")
	if err := os.MkdirAll(gdmDir, 0777); err != nil {
		return nil, err
	}

	dsm := storage.NewDiskStateManager(gdmDir)
	if err := dsm.WriteState(state, sous.User{}); err != nil {
		return nil, err
	}

	if err := doCMD(gdmDir, "git", "init"); err != nil {
		return nil, err
	}
	if err := doCMD(gdmDir, "git", "config", "user.name", "Sous Test"); err != nil {
		return nil, err
	}
	if err := doCMD(gdmDir, "git", "config", "user.email", "soustest@example.com"); err != nil {
		return nil, err
	}
	if err := doCMD(gdmDir, "git", "add", "."); err != nil {
		return nil, err
	}
	if err := doCMD(gdmDir, "git", "commit", "-a", "-m", "initial commit"); err != nil {
		return nil, err
	}

	gdmDir2 := path.Join(baseDir, "remote-gdm")
	if err := doCMD(gdmDir+"/..", "git", "clone", "--bare", gdmDir, gdmDir2); err != nil {
		return nil, err
	}

	count := len(state.Defs.Clusters)

	instances := make([]*Instance, count)
	for i := 0; i < count; i++ {
		clusterName := state.Defs.Clusters.Names()[i]
		inst, err := makeInstance(i, clusterName, baseDir)
		if err != nil {
			return nil, errors.Wrapf(err, "making test instance %d", i)
		}
		instances[i] = inst
	}
	return &TestCluster{
		BaseDir:      baseDir,
		RemoteGDMDir: gdmDir2,
		Count:        count,
		Instances:    instances,
	}, nil
}

func (c *TestCluster) Configure(envDesc desc.EnvDesc) error {
	siblingURLs := make(map[string]string, c.Count)
	for _, i := range c.Instances {
		siblingURLs[i.ClusterName] = "http://" + i.Addr
	}
	for _, i := range c.Instances {
		config := &config.Config{
			StateLocation: i.StateDir,
			SiblingURLs:   siblingURLs,
			Docker: docker.Config{
				RegistryHost:       envDesc.RegistryName(),
				DatabaseDriver:     "sqlite3_sous" + i.ClusterName,
				DatabaseConnection: "file:dummy_" + i.ClusterName + ".db?mode=memory&cache=shared",
			},
			User: sous.User{
				Name:  "Sous Server " + i.ClusterName,
				Email: "sous-" + i.ClusterName + "@example.com",
			},
		}
		if err := i.Configure(config, c.RemoteGDMDir); err != nil {
			return errors.Wrapf(err, "configuring instance %d", i)
		}
	}
	return nil
}

func (c *TestCluster) Start(t *testing.T, sousBin string) error {
	for j, i := range c.Instances {
		if err := i.Start(t, sousBin); err != nil {
			return errors.Wrapf(err, "instance%d", j)
		}
	}
	return nil
}

func (i *Instance) Configure(config *config.Config, remoteGDMDir string) error {
	if err := os.MkdirAll(i.StateDir, 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(i.ConfigDir, 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(i.LogDir, 0777); err != nil {
		return err
	}
	y, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	gdmDir := i.StateDir
	if err := doCMD(gdmDir+"/..", "git", "clone", remoteGDMDir, gdmDir); err != nil {
		return err
	}
	if err := doCMD(gdmDir, "git", "config", "user.name",
		fmt.Sprintf("Sous Server %s", i.ClusterName)); err != nil {
		return err
	}
	if err := doCMD(gdmDir, "git", "config", "user.email",
		fmt.Sprintf("sous-%s@example.com", i.ClusterName)); err != nil {
		return err
	}

	configFile := path.Join(i.ConfigDir, "config.yaml")
	if err := ioutil.WriteFile(configFile, y, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func (i *Instance) RunCmd(t *testing.T, binPath string, args ...string) (*exec.Cmd, error) {

	cmd := exec.Command(binPath, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("SOUS_CONFIG_DIR=%s", i.ConfigDir))
	stderr, err := os.Create(path.Join(i.LogDir, "stderr"))
	if err != nil {
		return cmd, err
	}
	stdout, err := os.Create(path.Join(i.LogDir, "stdout"))
	if err != nil {
		return cmd, err
	}
	combined, err := os.Create(path.Join(i.LogDir, "combined"))
	if err != nil {
		return cmd, err
	}

	cmd.Stdout = io.MultiWriter(stdout, combined, os.Stdout)
	cmd.Stderr = io.MultiWriter(stderr, combined, os.Stderr)

	return cmd, cmd.Start()
}

func (i *Instance) Start(t *testing.T, binPath string) error {

	fmt.Fprintf(os.Stderr, "==> Instance %q config:\n", i.ClusterName)
	configCMD, err := i.RunCmd(t, binPath, "config")
	if err != nil {
		t.Fatalf("setting up 'sous config': %s", err)
	}
	if err := configCMD.Wait(); err != nil {
		t.Fatalf("running 'sous config': %s", err)
	}

	serverDebug := os.Getenv("SOUS_SERVER_DEBUG") == "true"
	cmd, err := i.RunCmd(t, binPath, "server", "-listen", i.Addr, "-cluster", i.ClusterName, fmt.Sprintf("-d=%t", serverDebug))
	if err != nil {
		return err
	}
	if cmd.Process == nil {
		panic("cmd.Process nil after cmd.Start")
	}

	i.Proc = cmd.Process
	writePID(t, i.Proc.Pid)
	return nil
}

const pidFile = "test-server-pids"

// testing os.ErrNotExist seems to not work in the majority of cases, at least on Darwin.
// TODO SS: Find out why...
func isNotExist(err error) bool {
	if err == nil {
		panic("cannot check nil error")
	}
	return err == os.ErrNotExist ||
		strings.Contains(err.Error(), "no such file or directory")
}

func stopPIDs(t *testing.T) {
	t.Helper()
	d, err := ioutil.ReadFile(pidFile)
	if err != nil {
		if isNotExist(err) {
			return
		}
		t.Fatalf("unable to read %q: %s", pidFile, err)
		return
	}
	pids := strings.Split(string(d), "\n")
	var failedPIDs []string
	for _, p := range pids {
		if len(p) == 0 {
			continue
		}
		parts := strings.Split(p, "\t")
		if len(parts) != 2 {
			t.Fatalf("%q corrupted: contains %s", pidFile, p)
		}
		tmp, executable := parts[0], parts[1]
		p = tmp
		pid, err := strconv.Atoi(p)
		if err != nil {
			t.Fatalf("%q corrupted: contains %q (not an int)", pidFile, p)
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			if err != os.ErrNotExist {
				t.Fatalf("cannot find proc %d: %s", pid, err)
			}
		}
		psProc, err := ps.FindProcess(pid)
		if err != nil {
			t.Fatalf("cannot inspect proc %d: %s", pid, err)
		}
		if psProc == nil {
			t.Logf("skipping cleanup of %d (already stopped)", pid)
			continue
		}

		if psProc.Executable() != executable {
			t.Logf("not killing process %s; it is %q not %q", p, psProc.Executable(), executable)
			continue
		}
		if err := proc.Kill(); err != nil {
			failedPIDs = append(failedPIDs, p)
			t.Errorf("failed to stop process %d: %s", pid, err)
		}
	}
	if len(failedPIDs) != 0 {
		err := ioutil.WriteFile(pidFile, []byte(strings.Join(failedPIDs, "\n")), 0777)
		if err != nil {
			t.Fatalf("Failed to track failed PIDs %s: %s", strings.Join(failedPIDs, ", "), err)
		}
	} else {
		os.Remove(pidFile)
	}
}

func closeFile(t *testing.T, f *os.File) {
	if err := f.Close(); err != nil {
		t.Errorf("failed to close %s: %s", pidFile, err)
	}
}

func writePID(t *testing.T, pid int) {
	psProc, err := ps.FindProcess(pid)
	if err != nil {
		t.Fatalf("cannot inspect proc %d: %s", pid, err)
	}

	var f *os.File
	if s, err := os.Stat(pidFile); err != nil {
		if !isNotExist(err) {
			t.Fatalf("could not stat %q: %s", pidFile, err)
			return
		}
		if s != nil && s.IsDir() {
			t.Fatalf("cannot write to file %q: it's a directory", pidFile)
		}
		f, err = os.Create(pidFile)
		defer closeFile(t, f)
		if err != nil {
			t.Fatalf("could not create %q: %s", pidFile, err)
		}
	}
	if f == nil {
		var err error
		f, err = os.OpenFile(pidFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			t.Fatalf("could not open %q: %s", pidFile, err)
			return
		}
		defer closeFile(t, f)
	}
	if _, err := fmt.Fprintf(f, "%d\t%s\n", pid, psProc.Executable()); err != nil {
		t.Fatalf("could not write PID %d (exe %s) to file %q: %s",
			pid, psProc.Executable(), pidFile, err)
	}
}

func makeInstance(i int, clusterName, baseDir string) (*Instance, error) {
	baseDir = path.Join(baseDir, fmt.Sprintf("instance%d", i))
	stateDir := path.Join(baseDir, "state")
	configDir := path.Join(baseDir, "config")
	logDir := path.Join(baseDir, "logs")
	return &Instance{
		Addr:        fmt.Sprintf("127.0.0.1:%d", 6600+i),
		ClusterName: clusterName,
		StateDir:    stateDir,
		ConfigDir:   configDir,
		LogDir:      logDir,
	}, nil
}

func (c *TestClient) Cmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cmd := mkCMD(c.Dir, c.BinPath, args...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("SOUS_CONFIG_DIR=%s", c.ConfigDir))
	return cmd
}

func (c *TestClient) Run(t *testing.T, args ...string) (string, error) {
	cmd := c.Cmd(t, args...)
	fmt.Fprintf(os.Stderr, "SOUS_CONFIG_DIR = %q\n", c.ConfigDir)
	fmt.Fprintf(os.Stderr, "running sous in %q: %s\n", c.Dir, args)
	// Add quotes to args with spaces for printing.
	for i, a := range args {
		if strings.Contains(a, " ") {
			args[i] = `"` + a + `"`
		}
	}
	out := &bytes.Buffer{}
	cmd.Stdout = io.MultiWriter(os.Stdout, out)
	cmd.Stderr = os.Stderr
	fmt.Fprintf(os.Stderr, "==> sous %s\n", strings.Join(args, " "))
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return out.String(), err
}

func TestSousNewdeploy(t *testing.T) {
	f := setupEnv(t)

	// defer f.Stop(t)

	docekrfile := `FROM alpine
CMD if [ -z "$T" ]; then T=2; fi; echo -n "Sleeping ${T}s..."; sleep $T; echo "Done"; echo "Listening on :$PORT0"; while true; do echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0; done
`

	projectDir := path.Join(os.TempDir(), "project1")
	if err := os.RemoveAll(projectDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectDir, 0777); err != nil {
		t.Fatal(err)
	}
	dockerfilePath := path.Join(projectDir, "Dockerfile")
	if err := ioutil.WriteFile(dockerfilePath, []byte(dockerfile), 0777); err != nil {
		t.Fatal(err)
	}

	if err := func() error {
		gdmDir := projectDir
		if err := doCMD(gdmDir, "git", "init"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "config", "user.name", "Sous User"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "config", "user.email", "sous-user@example.com"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "add", "Dockerfile"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "commit", "-am", "add Dockerfile"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "remote", "add", "origin", "git@github.com:opentable/bogus/repo.git"); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		t.Fatal(err)
	}

	sous := f.Client

	if _, err := sous.Run(t, "version"); err != nil {
		t.Fatal(err)
	}

	// sous init
	sous.Dir = projectDir
	if _, err := sous.Run(t, "init"); err != nil {
		t.Fatal(err)
	}

	// sous manifest get > manifest
	manifestGetCmd := sous.Cmd(t, "manifest", "get")
	manifestBytes, err := manifestGetCmd.Output()
	if err != nil {
		//output, err := manifestGetCmd.CombinedOutput()
		//		if err != nil {
		t.Fatalf("sous manifest get failed: %s; output:\n%s", err, string(manifestBytes))
		//}
		//t.Fatalf("sous manifest get weirdly failed then succeeded")
	}

	fmt.Println(string(manifestBytes))

	// edit manifest
	manifest := strings.Replace(string(manifestBytes), "NumInstances: 0", "NumInstances: 1", -1)

	fmt.Print(manifest)

	// sous manifest set < manifest
	manifestSetCmd := sous.Cmd(t, "manifest", "set")
	manifestSetCmd.Stdin = ioutil.NopCloser(bytes.NewReader([]byte(manifest)))

	if out, err := manifestSetCmd.CombinedOutput(); err != nil {
		t.Fatalf("manifest set failed: %s; output:\n%s", err, out)
	}

	if _, err := sous.Run(t, "build", "-tag", "1.2.3"); err != nil {
		t.Fatal(err)
	}

	if _, err := sous.Run(t, "newdeploy", "-d", "-cluster", "cluster1", "-tag", "1.2.3"); err != nil {
		t.Fatal(err)
	}

	panic("DONE")

}
