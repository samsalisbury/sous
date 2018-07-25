package smoke

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/yaml"
)

type sousServer struct {
	*Service
	Addr                string
	StateDir, ConfigDir string
	ClusterName         string
	// Num is the instance number for display purposes.
	Num int
}

func makeInstance(t *testing.T, binPath string, i int, clusterName, baseDir, addr string, finished <-chan struct{}) (*sousServer, error) {
	baseDir = path.Join(baseDir, fmt.Sprintf("instance%d", i+1))
	stateDir := path.Join(baseDir, "state")

	num := i + 1

	name := fmt.Sprintf("instance%d", num)

	bin := NewBin(binPath, name, baseDir, finished)
	bin.Env["SOUS_CONFIG_DIR"] = bin.ConfigDir

	service := NewService(bin)

	return &sousServer{
		Service:     service,
		Addr:        addr,
		ClusterName: clusterName,
		StateDir:    stateDir,
		Num:         num,
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

	i.Service.Configure(filemap.FileMap{
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
	i.MustRun(t, "config", nil)

	serverDebug := os.Getenv("SOUS_SERVER_DEBUG") == "true"

	i.Service.Start(t, "server", nil, "-listen", i.Addr, "-cluster", i.ClusterName, "autoresolver=false", fmt.Sprintf("-d=%t", serverDebug))

}

const pidFile = "test-server-pids"
