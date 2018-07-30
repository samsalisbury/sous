package smoke

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
)

type fixtureConfig struct {
	matrix      matrixCombo
	startState  *sous.State
	envDesc     desc.EnvDesc
	singularity *testSingularity
	// TODO SS: Remove this in favour of funcs that isolate for particular
	// pieces of data.
	clusterSuffix string
}

type matrixCombo struct {
	dbPrimary bool
	projects  projectList
}

// matrix returns the defined sous smoke test matrix.
func matrix() Matrix {
	m := New()
	m.AddDimension("store", "GDM storage to use", map[string]interface{}{
		"db":  true,
		"git": false,
	})
	m.AddDimension("project", "type of project to build", map[string]interface{}{
		"simple": projects.SingleDockerfile,
		"split":  projects.SplitBuild,
	})
	return m
}

func makeFixtureConfig(t *testing.T, c Scenario) fixtureConfig {
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
		matrix:        makeMatrixCombo(c),
		envDesc:       envDesc,
		clusterSuffix: clusterSuffix,
		singularity:   s9y,
		startState:    state,
	}
}

func makeMatrixCombo(c Scenario) matrixCombo {
	m := c.Map()
	return matrixCombo{
		dbPrimary: m["store"].(bool),
		projects:  m["project"].(projectList),
	}
}
