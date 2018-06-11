//+build smoke

package smoke

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
)

type TestFixture struct {
	EnvDesc     desc.EnvDesc
	Cluster     TestBunchOfSousServers
	Client      *TestClient
	BaseDir     string
	Singularity *Singularity
	// ClusterSuffix is used to add a suffix to each generated cluster name.
	// This can be used to segregate parallel tests.
	ClusterSuffix string
	Parent        *ParallelTestFixture
	TestName      string
	UserEmail     string
	knownToFail   bool
}

func newTestFixture(t *testing.T, parent *ParallelTestFixture, nextAddr func() string, fcfg fixtureConfig) TestFixture {
	t.Helper()
	t.Parallel()
	if testing.Short() {
		t.Skipf("-short flag present")
	}
	sousBin := getSousBin(t)
	envDesc := getEnvDesc(t)
	baseDir := getDataDir(t)

	clusterSuffix := strings.Replace(t.Name(), "/", "_", -1)
	fmt.Fprintf(os.Stdout, "Cluster suffix: %s", clusterSuffix)

	singularity := NewSingularity(envDesc.SingularityURL())
	singularity.ClusterSuffix = clusterSuffix

	state := sous.StateFixture(sous.StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
		ClusterSuffix: clusterSuffix,
	})

	addURLsToState(state, envDesc)

	fcfg.startState = state

	c, err := newBunchOfSousServers(t, baseDir, nextAddr, fcfg)
	if err != nil {
		t.Fatalf("setting up test cluster: %s", err)
	}

	if err := c.Configure(t, envDesc, fcfg); err != nil {
		t.Fatalf("configuring test cluster: %s", err)
	}

	if err := c.Start(t, sousBin); err != nil {
		t.Fatalf("starting test cluster: %s", err)
	}

	client := makeClient(baseDir, sousBin)
	primaryServer := "http://" + c.Instances[0].Addr
	userEmail := "sous_client1@example.com"
	if err := client.Configure(primaryServer, envDesc.RegistryName(), userEmail); err != nil {
		t.Fatal(err)
	}

	return TestFixture{
		Cluster:       *c,
		Client:        client,
		BaseDir:       baseDir,
		Singularity:   singularity,
		ClusterSuffix: clusterSuffix,
		Parent:        parent,
		TestName:      t.Name(),
		UserEmail:     userEmail,
	}
}

func (f *TestFixture) Stop(t *testing.T) {
	t.Helper()
	f.Cluster.Stop(t)
}

func (f *TestFixture) ReportStatus(t *testing.T) {
	t.Helper()
	f.Parent.recordTestStatus(t)
}

func (f *TestFixture) KnownToFailHere(t *testing.T) {
	t.Helper()
	const skipKnownFailuresEnvVar = "EXCLUDE_KNOWN_FAILING_TESTS"
	if os.Getenv(skipKnownFailuresEnvVar) == "YES" {
		f.knownToFail = true
		t.Skipf("This test is known to fail and you set %s=YES",
			skipKnownFailuresEnvVar)
	}
}
