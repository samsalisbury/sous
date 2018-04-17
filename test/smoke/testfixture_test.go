//+build smoke

package smoke

import (
	"testing"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
)

type Fixture struct {
	EnvDesc     desc.EnvDesc
	Cluster     TestCluster
	Client      TestClient
	BaseDir     string
	Singularity *Singularity
}

func setupEnv(t *testing.T) Fixture {
	t.Helper()
	if testing.Short() {
		t.Skipf("-short flag present")
	}
	stopPIDs(t)
	sousBin := getSousBin(t)
	envDesc := getEnvDesc(t)
	baseDir := getDataDir(t)

	singularity := NewSingularity(envDesc.SingularityURL())

	singularity.Reset(t)

	state := sous.StateFixture(sous.StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
	})

	addURLsToState(state, envDesc)

	c, err := newSmokeTestFixture(state, baseDir)
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
		Cluster:     *c,
		Client:      client,
		BaseDir:     baseDir,
		Singularity: singularity,
	}
}

func (f *Fixture) Stop(t *testing.T) {
	t.Helper()
	f.Cluster.Stop(t)
}
