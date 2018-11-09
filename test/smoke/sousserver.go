package smoke

import (
	"fmt"
	"net"
	"os"
	"path"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
)

type sousServer struct {
	*Service
	Addr                string
	StateDir, ConfigDir string
	ClusterName         string
	// Num is the instance number for display purposes.
	Num int
}

func makeInstance(t *testing.T, f fixtureConfig, binPath string, i int, clusterName, addr string) (*sousServer, error) {
	num := i + 1

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "getting port")
	}
	name := fmt.Sprintf("server%d-%s", num, port)

	bin := f.newBin(t, binPath, name)

	bin.Env["SOUS_CONFIG_DIR"] = bin.ConfigDir
	bin.Env["SOUS_BUILD_NOPULL"] = "YES"
	addGitEnvVars(bin.Env)

	service := NewService(bin)

	stateDir := path.Join(bin.BaseDir, "state")

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

func (i *sousServer) configure(t *testing.T, f fixtureConfig, config *config.Config, remoteGDMDir string) error {

	// TODO SS: Seed DB only when test starts.
	if err := seedDB(config, f.InitialState); err != nil {
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

	gdmDir := f.newEmptyDir(i.StateDir)
	g := newGitClient(t, f, fmt.Sprintf("server%d", i.Num))
	g.CD(gdmDir)
	g.cloneIntoCurrentDir(t, f, gitRepoSpec{
		OriginURL: remoteGDMDir,
		UserName:  fmt.Sprintf("Sous Server %s", i.ClusterName),
		UserEmail: fmt.Sprintf("sous-%s@example.com", i.ClusterName),
	})

	return nil
}

func (i *sousServer) Start(t *testing.T) {
	if !quiet() {
		fmt.Fprintf(os.Stderr, "==> Instance %q config:\n", i.ClusterName)
	}

	serverDebug := os.Getenv("SOUS_SERVER_DEBUG") == "true"

	i.Service.Start(t, "server", nil, "-listen", i.Addr, "-cluster", i.ClusterName, "-autoresolver=false", fmt.Sprintf("-d=%t", serverDebug))

}

const pidFile = "test-server-pids"
