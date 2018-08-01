package smoke

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/testmatrix"
)

type fixtureConfig struct {
	scenario    scenario
	startState  *sous.State
	envDesc     desc.EnvDesc
	singularity *testSingularity
	// TODO SS: Remove this in favour of funcs that isolate for particular
	// pieces of data.
	clusterSuffix string
}

func makeFixtureConfig(t *testing.T, c testmatrix.Scenario) fixtureConfig {
	envDesc := getEnvDesc()
	clusterSuffix := strings.Replace(t.Name(), "/", "_", -1)
	fmt.Fprintf(os.Stdout, "Cluster suffix: %s", clusterSuffix)
	s9y := newSingularity(envDesc.SingularityURL())
	s9y.ClusterSuffix = clusterSuffix
	state := sous.StateFixture(sous.StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
		ClusterSuffix: clusterSuffix,
	})
	addURLsToState(state, envDesc)
	return fixtureConfig{
		scenario:      unwrapScenario(c),
		envDesc:       envDesc,
		clusterSuffix: clusterSuffix,
		singularity:   s9y,
		startState:    state,
	}
}
