//+build smoke

package smoke

import (
	"os"
	"path"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

type TestBunchOfSousServers struct {
	BaseDir      string
	RemoteGDMDir string
	Count        int
	Instances    []*Instance
}

func newBunchOfSousServers(t *testing.T, baseDir string, nextFreeAddr func() string, fcfg fixtureConfig) (*TestBunchOfSousServers, error) {
	if err := os.MkdirAll(baseDir, 0777); err != nil {
		return nil, err
	}

	state := fcfg.startState

	gdmDir := path.Join(baseDir, "remote-gdm")
	if err := createRemoteGDM(gdmDir, state); err != nil {
		return nil, err
	}

	count := len(state.Defs.Clusters)
	instances := make([]*Instance, count)
	for i := 0; i < count; i++ {
		clusterName := state.Defs.Clusters.Names()[i]
		inst, err := makeInstance(t, i, clusterName, baseDir, nextFreeAddr())
		if err != nil {
			return nil, errors.Wrapf(err, "making test instance %d", i)
		}
		instances[i] = inst
	}
	return &TestBunchOfSousServers{
		BaseDir:      baseDir,
		RemoteGDMDir: gdmDir,
		Count:        count,
		Instances:    instances,
	}, nil
}

func createRemoteGDM(gdmDir string, state *sous.State) error {

	gdmDir2 := gdmDir
	gdmDir = gdmDir + "-temp"

	if err := os.MkdirAll(gdmDir, 0777); err != nil {
		return err
	}

	dsm := storage.NewDiskStateManager(gdmDir, logging.SilentLogSet())
	if err := dsm.WriteState(state, sous.User{}); err != nil {
		return err
	}

	if err := doCMD(gdmDir, "git", "init"); err != nil {
		return err
	}
	if err := doCMD(gdmDir, "git", "config", "user.name", "Sous Test"); err != nil {
		return err
	}
	if err := doCMD(gdmDir, "git", "config", "user.email", "soustest@example.com"); err != nil {
		return err
	}
	if err := doCMD(gdmDir, "git", "add", "."); err != nil {
		return err
	}
	if err := doCMD(gdmDir, "git", "commit", "-a", "-m", "initial commit"); err != nil {
		return err
	}

	if err := doCMD(gdmDir+"/..", "git", "clone", "--bare", gdmDir, gdmDir2); err != nil {
		return err
	}
	return nil
}

func (c *TestBunchOfSousServers) Configure(t *testing.T, envDesc desc.EnvDesc, fcfg fixtureConfig) error {
	siblingURLs := make(map[string]string, c.Count)
	for _, i := range c.Instances {
		siblingURLs[i.ClusterName] = "http://" + i.Addr
	}

	dbport := "6543"
	if np, set := os.LookupEnv("PGPORT"); set {
		dbport = np
	}

	host := "localhost"
	if h, set := os.LookupEnv("PGHOST"); set {
		host = h
	}

	for n, i := range c.Instances {
		sous.SetupDB(t, n)
		dbname := sous.DBNameForTest(t, n)
		config := &config.Config{
			StateLocation: i.StateDir,
			SiblingURLs:   siblingURLs,
			Database: storage.PostgresConfig{
				User:   "postgres",
				DBName: dbname,
				Host:   host,
				Port:   dbport,
			},
			DatabasePrimary: fcfg.dbPrimary,
			Docker: docker.Config{
				RegistryHost: envDesc.RegistryName(),
				//DatabaseConnection: "file:dummy_" + i.ClusterName + ".db?mode=memory&cache=shared",
			},
			User: sous.User{
				Name:  "Sous Server " + i.ClusterName,
				Email: "sous-" + i.ClusterName + "@example.com",
			},
		}
		config.Logging.Basic.Level = "debug"
		if err := i.Configure(config, c.RemoteGDMDir, fcfg); err != nil {
			return errors.Wrapf(err, "configuring instance %d", i)
		}
	}
	return nil
}

func (c *TestBunchOfSousServers) Start(t *testing.T, sousBin string) error {
	for j, i := range c.Instances {
		if err := i.Start(t, sousBin); err != nil {
			return errors.Wrapf(err, "instance%d", j)
		}
	}
	return nil
}
func (c *TestBunchOfSousServers) Stop(t *testing.T) {
	t.Helper()
	stopPIDs(t)
	for n := range c.Instances {
		sous.ReleaseDB(t, n)
	}
}
