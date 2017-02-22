package singularity

import (
	"fmt"

	sous "github.com/opentable/sous/lib"
)

// A testFixture represents a state of the world for tests to run in.
//
// It provides functions that make it easy to construct a consistent
// milieu in which tests can be run. The strategy for writing tests
// with this is to construct a healthy and consistent world, and then
// to introduce specific flaws against which tests can be written.
type testFixture struct {
	Singularities map[string]*testSingularity
	Registry      *testRegistry
	Clusters      sous.Clusters
}

func (tf *testFixture) DeployReaderFactory(c *sous.Cluster) DeployReader {
	return &testDeployReader{Fixture: tf}
}

func testImageName(repo, offset, tag string) string {
	return fmt.Sprintf("docker.mycompany.com/%s%s:%s", repo, offset, tag)
}

// AddSingularity adds a singularity if none exist for baseURL. It returns
// the one that already existed, or the new one created.
func (tf *testFixture) AddSingularity(baseURL string) *testSingularity {
	if tf.Singularities == nil {
		tf.Singularities = map[string]*testSingularity{}
	}
	if s, ok := tf.Singularities[baseURL]; ok {
		return s
	}
	singularity := &testSingularity{
		Parent:  tf,
		BaseURL: baseURL,
	}
	tf.Singularities[baseURL] = singularity
	return singularity
}
