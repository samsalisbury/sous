//+build smoke

package smoke

import (
	"testing"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
)

type Fixture struct {
	EnvDesc desc.EnvDesc
	Cluster TestCluster
	Client  TestClient
	BaseDir string
}

func setupEnv(t *testing.T, testName string) Fixture {
	t.Helper()
	if testing.Short() {
		t.Skipf("-short flag present")
	}
	stopPIDs(t)
	sousBin := getSousBin(t)
	envDesc := getEnvDesc(t)
	baseDir := getDataDir(t, testName)

	resetSingularity(t, envDesc.SingularityURL())

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
		Cluster: *c,
		Client:  client,
		BaseDir: baseDir,
	}
}
