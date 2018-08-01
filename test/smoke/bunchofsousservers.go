package smoke

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

type bunchOfSousServers struct {
	BaseDir      string
	RemoteGDMDir string
	Count        int
	Instances    []*sousServer
	Stop         func() error
}

func newBunchOfSousServers(t *testing.T, f fixtureConfig) (*bunchOfSousServers, error) {
	// TODO SS: This should have already happened..
	if err := os.MkdirAll(f.BaseDir, 0777); err != nil {
		return nil, err
	}

	gdmDir := path.Join(f.BaseDir, "remote-gdm")
	if err := createRemoteGDM(gdmDir, f.InitialState); err != nil {
		return nil, err
	}

	binPath := sousBin

	state := f.InitialState

	count := len(state.Defs.Clusters)
	instances := make([]*sousServer, count)
	addrs := freePortAddrs("127.0.0.1", count)
	for i := 0; i < count; i++ {
		clusterName := state.Defs.Clusters.Names()[i]
		inst, err := makeInstance(t, binPath, i, clusterName, f.BaseDir, addrs[i], f.Finished)
		if err != nil {
			return nil, errors.Wrapf(err, "making test instance %d", i)
		}
		instances[i] = inst
	}
	return &bunchOfSousServers{
		BaseDir:      f.BaseDir,
		RemoteGDMDir: gdmDir,
		Count:        count,
		Instances:    instances,
		Stop: func() error {
			return fmt.Errorf("cannot stop bunch of sous servers (not started)")
		},
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

func (c *bunchOfSousServers) configure(t *testing.T, f fixtureConfig) error {
	t.Helper()
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
		dbname := sous.DBNameForTest(t, n)

		if _, err := sous.SetupDBNamed(t, dbname); err != nil {
			rtLog("%s:db:%s> create failed: %s", i.ID(), dbname, err)
			t.Fatalf("create database failed: %s", err)
		}
		rtLog("%s:db:%s> created", i.ID(), dbname)

		config := &config.Config{
			StateLocation: i.StateDir,
			SiblingURLs:   siblingURLs,
			Database: storage.PostgresConfig{
				User:   "postgres",
				DBName: dbname,
				Host:   host,
				Port:   dbport,
			},
			DatabasePrimary: f.Scenario.dbPrimary,
			Docker: docker.Config{
				RegistryHost: f.EnvDesc.RegistryName(),
			},
			User: sous.User{
				Name:  "Sous Server " + i.ClusterName,
				Email: "sous-" + i.ClusterName + "@example.com",
			},
		}
		config.Logging.Basic.Level = "debug"
		if err := i.configure(config, c.RemoteGDMDir, f); err != nil {
			return errors.Wrapf(err, "configuring instance %d", i)
		}
	}
	return nil
}

func (c *bunchOfSousServers) Start(t *testing.T) {
	t.Helper()
	var started []*sousServer
	// Set the stop func first in case starting returns early.
	c.Stop = func() error {
		var errs []string
		for j, i := range started {
			if err := i.Stop(); err != nil {
				errs = append(errs, fmt.Sprintf(`"could not stop instance%d: %s"`, j, err))
			}
		}
		if len(errs) == 0 {
			return nil
		}
		return fmt.Errorf("could not stop all instances: %s", strings.Join(errs, ", "))
	}
	for _, i := range c.Instances {
		i.Start(t)
		// Note: the value of started is only used in the closure above.
		started = append(started, i)
	}
}
