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
	"github.com/pkg/errors"
)

type TestBunchOfSousServers struct {
	BaseDir      string
	RemoteGDMDir string
	Count        int
	Instances    []*Instance
}

func newBunchOfSousServers(t *testing.T, state *sous.State, baseDir string, nextFreeAddr func() string) (*TestBunchOfSousServers, error) {
	if err := os.MkdirAll(baseDir, 0777); err != nil {
		return nil, err
	}

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

	dsm := storage.NewDiskStateManager(gdmDir)
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

func (c *TestBunchOfSousServers) Configure(envDesc desc.EnvDesc) error {
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
}
