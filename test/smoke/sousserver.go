package smoke

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/firsterr"
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
	// Name is derived from Num.
	Name string
}

func makeInstance(t *testing.T, f *fixtureConfig, binPath string, i int, clusterName, addr string) (*sousServer, error) {
	num := i + 1

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "getting port")
	}
	name := fmt.Sprintf("server%d-%s", num, port)
	name = fmt.Sprintf("server%d", num)

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
		Name:        name,
	}, nil
}

func seedDB(db *sql.DB, state *sous.State) error {
	mgr := storage.NewPostgresStateManager(db, logging.SilentLogSet())

	return mgr.WriteState(state, sous.User{})
}

func (i *sousServer) initDB(t *testing.T) storage.PostgresConfig {
	dbport := "6543"
	if np, set := os.LookupEnv("PGPORT"); set {
		dbport = np
	}
	host := "localhost"
	if h, set := os.LookupEnv("PGHOST"); set {
		host = h
	}
	dbname := sous.DBNameForTest(t, i.Num)
	if _, err := sous.SetupDBNamed(t, dbname); err != nil {
		rtLog("%s:db:%s> create failed: %s", i.ID(), dbname, err)
		t.Fatalf("create database failed: %s", err)
	}
	rtLog("%s:db:%s> created", i.ID(), dbname)
	return storage.PostgresConfig{
		User:   "postgres",
		DBName: dbname,
		Host:   host,
		Port:   dbport,
	}
}

func (i *sousServer) initConfigAndState(t *testing.T, f *fixtureConfig, siblingURLs map[string]string) (*config.Config, error) {

	pgConfig := i.initDB(t)

	config := &config.Config{
		StateLocation:   i.StateDir,
		SiblingURLs:     siblingURLs,
		Database:        pgConfig,
		DatabasePrimary: f.DBPrimary,
		Docker: docker.Config{
			RegistryHost: f.EnvDesc.RegistryName(),
		},
		User: sous.User{
			Name:  "Sous Server " + i.ClusterName,
			Email: "sous-" + i.ClusterName + "@example.com",
		},
	}

	config.Logging.Basic.Level = "debug"

	return config, firsterr.Parallel().Set(
		func(e *error) {
			if !f.shouldHavePostgresState() {
				return
			}
			var db *sql.DB
			db, *e = config.Database.DB()
			if *e == nil {
				*e = seedDB(db, f.InitialState)
			}
		},
		func(e *error) {
			if !f.shouldHaveGitState() {
				return
			}
			*e = i.initializeGitState(t, f)
		},
	)
}

func (i *sousServer) initializeGitState(t *testing.T, f *fixtureConfig) error {
	if err := os.MkdirAll(i.StateDir, 0777); err != nil {
		return err
	}
	gdmDir := f.newEmptyDir(i.StateDir)
	g := newGitClient(t, f, i.Name)
	g.CD(gdmDir)
	g.cloneIntoCurrentDir(t, gitRepoSpec{
		OriginURL: f.remoteGDM(t),
		UserName:  fmt.Sprintf("Sous Server %s", i.ClusterName),
		UserEmail: fmt.Sprintf("sous-%s@example.com", i.ClusterName),
	})
	return nil
}

func (i *sousServer) configure(t *testing.T, f *fixtureConfig, siblingURLs map[string]string) error {

	config, err := i.initConfigAndState(t, f, siblingURLs)
	if err != nil {
		return err
	}

	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	i.Service.Configure(filemap.FileMap{
		"config.yaml": string(configYAML),
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
