package smoke

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/yaml"
)

type sousServer struct {
	Bin
	Addr                string
	StateDir, ConfigDir string
	ClusterName         string
	Proc                *os.Process
	// Num is the instance number for display purposes.
	Num int
}

func makeInstance(t *testing.T, binPath string, i int, clusterName, baseDir, addr string, finished <-chan struct{}) (*sousServer, error) {
	baseDir = path.Join(baseDir, fmt.Sprintf("instance%d", i+1))
	stateDir := path.Join(baseDir, "state")

	name := fmt.Sprintf("instance%d", i)

	bin := NewBin(binPath, name, baseDir, finished)
	bin.Env["SOUS_CONFIG_DIR"] = bin.ConfigDir

	return &sousServer{
		Bin:         bin,
		Addr:        addr,
		ClusterName: clusterName,
		StateDir:    stateDir,
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

func (i *sousServer) configure(config *config.Config, remoteGDMDir string, fcfg fixtureConfig) error {
	if err := seedDB(config, fcfg.startState); err != nil {
		return err
	}

	if err := os.MkdirAll(i.StateDir, 0777); err != nil {
		return err
	}

	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	i.Bin.Configure(filemap.FileMap{
		"config.yaml": string(configYAML),
	})

	gdmDir := i.StateDir
	if err := doCMD(gdmDir+"/..", "git", "clone", remoteGDMDir, gdmDir); err != nil {
		return err
	}
	username := fmt.Sprintf("Sous Server %s", i.ClusterName)
	if err := doCMD(gdmDir, "git", "config", "user.name", username); err != nil {
		return err
	}
	email := fmt.Sprintf("sous-%s@example.com", i.ClusterName)
	if err := doCMD(gdmDir, "git", "config", "user.email", email); err != nil {
		return err
	}

	return nil
}

func (i *sousServer) Start(t *testing.T) {
	t.Helper()

	if !quiet() {
		fmt.Fprintf(os.Stderr, "==> Instance %q config:\n", i.ClusterName)
	}
	// Run 'sous config' to validate it.
	i.Bin.MustRun(t, "config", nil)

	serverDebug := os.Getenv("SOUS_SERVER_DEBUG") == "true"
	prepared := i.Bin.Command(t, "server", nil, "-listen", i.Addr, "-cluster", i.ClusterName, "autoresolver=false", fmt.Sprintf("-d=%t", serverDebug))

	cmd := prepared.Cmd
	if err := cmd.Start(); err != nil {
		t.Fatalf("error starting server %q: %s", i.InstanceName, err)
	}

	if cmd.Process == nil {
		panic("cmd.Process nil after cmd.Start")
	}

	go func() {
		id := fmt.Sprintf("%s:%s", t.Name(), i.InstanceName)

		var ps *os.ProcessState
		select {
		// In this case the process ended before the test finished.
		case err := <-func() <-chan error {
			var err error
			c := make(chan error, 1)
			go func() {
				ps, err = cmd.Process.Wait()
				c <- err
			}()
			return c
		}():
			if err != nil {
				rtLog("SERVER CRASHED: %s: %s; process state: %s", id, err, ps)
				return
			}
			if !ps.Exited() {
				// NOTE SS: This condition should not be possible, since after
				// calling Wait, the process should have exited. But it hasn't.
				rtLog("SERVER DID NOT EXIT: %s", id)
				return
			}
			if ps.Success() {
				rtLog("SERVER STOPPED: %s (exit code 0)", id)
			}
			// TODO SS: Dump log tail here as well for analysis.
			rtLog("SERVER CRASHED: %s; logs follow:", id)
			i.DumpTail(t, 50)
		// In this case the process is still running.
		case <-i.TestFinished:
			rtLog("OK: SERVER STILL RUNNING AFTER TEST %s", id)
			// Do nothing.
		}
	}()

	i.Proc = cmd.Process
	writePID(t, i.Proc.Pid)
}

func (i *sousServer) DumpTail(t *testing.T, n int) {
	path := filepath.Join(i.LogDir, "combined")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		rtLog("ERROR unable to read log file %s: %s", path, err)
	}
	lines := strings.Split(string(b), "\n")
	out := strings.Join(lines[len(lines)-n:], "\n") + "\n"
	prefix := fmt.Sprintf("%s:%s:combined> ", t.Name(), i.InstanceName)
	outPipe, err := prefixedPipe(prefix)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(outPipe, out)
}

func (i *sousServer) Stop() error {
	if i.Proc == nil {
		return fmt.Errorf("cannot stop instance %q (not started)", i.Num)
	}
	if err := i.Proc.Kill(); err != nil {
		return fmt.Errorf("cannot kill instance %q: %s", i.Num, err)
	}
	return nil
}

const pidFile = "test-server-pids"
