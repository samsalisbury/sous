//+build smoke

package smoke

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/yaml"
)

type Instance struct {
	Addr                string
	StateDir, ConfigDir string
	ClusterName         string
	Proc                *os.Process
	LogDir              string
	// Num is the instance number for display purposes.
	Num int
}

func makeInstance(t *testing.T, i int, clusterName, baseDir, addr string) (*Instance, error) {
	baseDir = path.Join(baseDir, fmt.Sprintf("instance%d", i+1))
	stateDir := path.Join(baseDir, "state")
	configDir := path.Join(baseDir, "config")
	logDir := path.Join(baseDir, "logs")

	return &Instance{
		Addr:        addr,
		ClusterName: clusterName,
		StateDir:    stateDir,
		ConfigDir:   configDir,
		LogDir:      logDir,
		Num:         i + 1,
	}, nil
}

func seedDB(config *config.Config, state *sous.State) error {
	db, err := config.Database.DB()
	if err != nil {
		return err
	}
	mgr := storage.NewPostgresStateManager(db, logging.SilentLogSet())

	return mgr.WriteState(state, sous.User{})
}

func (i *Instance) Configure(config *config.Config, remoteGDMDir string, fcfg fixtureConfig) error {
	if err := seedDB(config, fcfg.startState); err != nil {
		return err
	}

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
	stderrFile, err := os.Create(path.Join(i.LogDir, "stderr"))
	if err != nil {
		return cmd, err
	}
	stdoutFile, err := os.Create(path.Join(i.LogDir, "stdout"))
	if err != nil {
		return cmd, err
	}
	combinedFile, err := os.Create(path.Join(i.LogDir, "combined"))
	if err != nil {
		return cmd, err
	}

	//stdout, stderr := prefixWithTestName(t, fmt.Sprintf("instance%d", i.Num))
	stdout, stderr := ioutil.Discard, ioutil.Discard

	cmd.Stdout = io.MultiWriter(stdout, stdoutFile, combinedFile)
	cmd.Stderr = io.MultiWriter(stderr, stderrFile, combinedFile)

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
