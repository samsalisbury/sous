//+build smoke

package smoke

import (
	"fmt"
	"os"
	"strings"
	"sync"
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
}

func resetSingularity(t *testing.T) {
	envDesc := getEnvDesc(t)
	singularity := NewSingularity(envDesc.SingularityURL())
	singularity.Reset(t)
}

var testNames = map[string]struct{}{}
var testNamesMu sync.Mutex

func assertUniqueTestName(t *testing.T) {
	name := t.Name()
	testNamesMu.Lock()
	defer testNamesMu.Unlock()
	if _, ok := testNames[name]; ok {
		t.Fatalf("duplicate test name: %q", name)
	}
	testNames[name] = struct{}{}
}

func newTestFixture(t *testing.T, nextAddr func() string) TestFixture {
	t.Helper()
	t.Parallel()
	if testing.Short() {
		t.Skipf("-short flag present")
	}
	sousBin := getSousBin(t)
	envDesc := getEnvDesc(t)
	baseDir := getDataDir(t)

	assertUniqueTestName(t)
	clusterSuffix := strings.Replace(t.Name(), "/", "_", -1)
	fmt.Fprintf(os.Stdout, "Cluster suffix: %s", clusterSuffix)
	//t.Skipf("Cluster suffix: %q", clusterSuffix)

	singularity := NewSingularity(envDesc.SingularityURL())
	singularity.ClusterSuffix = clusterSuffix

	state := sous.StateFixture(sous.StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
		ClusterSuffix: clusterSuffix,
	})

	addURLsToState(state, envDesc)

	c, err := newBunchOfSousServers(t, state, baseDir, nextAddr)
	if err != nil {
		t.Fatalf("setting up test cluster: %s", err)
	}

	if err := c.Configure(t, envDesc); err != nil {
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

	return TestFixture{
		Cluster:       *c,
		Client:        client,
		BaseDir:       baseDir,
		Singularity:   singularity,
		ClusterSuffix: clusterSuffix,
	}
}

func (f *TestFixture) Stop(t *testing.T) {
	t.Helper()
	f.Cluster.Stop(t)
}
