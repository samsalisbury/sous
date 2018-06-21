package sous

import (
	"fmt"
	"strings"

	"github.com/samsalisbury/semv"
)

// StateFixtureOpts allows configuring StateFixture calls.
type StateFixtureOpts struct {
	ClusterCount  int
	ManifestCount int
	ClusterSuffix string
}

// DefaultStateFixture provides a dummy state for tests by calling
// StateFixture with the following options:
//
//  StateFixtureOpts{
//	  ClusterCount:  3,
//    ManifestCount: 3,
//  }
func DefaultStateFixture() *State {
	return StateFixture(StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
	})
}

// StateFixture provides a dummy state for tests.
func StateFixture(o StateFixtureOpts) *State {

	s := NewState()

	m := NewManifests()
	for manifestN := 0; manifestN < o.ManifestCount; manifestN++ {
		m.Add(&Manifest{
			Source: SourceLocation{
				Repo: fmt.Sprintf("github.com/user%d/repo%d", manifestN, manifestN),
				Dir:  fmt.Sprintf("dir%d", manifestN),
			},
			Flavor: fmt.Sprintf("flavor%d", manifestN),
			Owners: nil,
			Kind:   "http-service",
		})
	}

	c := Clusters{}
	// For each cluster add it to defs and add a deployment to each manifest.
	for clusterN := 0; clusterN < o.ClusterCount; clusterN++ {
		clusterName := fmt.Sprintf("cluster%d%s", clusterN, o.ClusterSuffix)
		c[clusterName] = &Cluster{
			Name:    clusterName,
			Kind:    "singularity",
			BaseURL: "127.0.0.1:5000",
			Env: map[string]Var{
				"CLUSTER_NAME": Var(clusterName),
			},
			Startup: Startup{
				SkipCheck:                 false,
				ConnectDelay:              30,
				Timeout:                   30,
				ConnectInterval:           10,
				CheckReadyProtocol:        "HTTP",
				CheckReadyURIPath:         "/health",
				CheckReadyFailureStatuses: []int{500},
				CheckReadyURITimeout:      2,
				CheckReadyInterval:        2,
				CheckReadyRetries:         256,
			},
			AllowedAdvisories: AllAdvisoryStrings(),
		}
		for _, mid := range m.Keys() {
			manifest, ok := m.Get(mid)
			if !ok {
				panic("Manifests.Keys returned a nonexistent key")
			}

			if manifest.Deployments == nil {
				manifest.Deployments = DeploySpecs{}
			}
			did := DeploymentID{ManifestID: mid, Cluster: clusterName}
			manifest.Deployments[clusterName] = DeploySpec{
				DeployConfig: DeployConfig{
					Resources: map[string]string{
						"cpus":   "0.1",
						"memory": "32",
						"ports":  "1",
					},
					Startup: Startup{
						CheckReadyProtocol: "HTTP",
					},
					Metadata: map[string]string{
						"": "",
					},
					Env: map[string]string{
						"": "",
					},
					NumInstances:         3,
					Volumes:              nil,
					Schedule:             "",
					SingularityRequestID: makeTestFixtureSingularityReqID(did),
				},
				Version: semv.MustParse("1.0.0"),
			}
		}
	}

	s.Defs = Defs{
		DockerRepo: "docker.example.com",
		Clusters:   c,
	}
	s.Manifests = m

	return s
}

func makeTestFixtureSingularityReqID(did DeploymentID) string {
	repo := strings.Split(did.ManifestID.Source.Repo, "/")
	dir := strings.Replace(did.ManifestID.Source.Dir, "/", "-", -1)
	if dir != "" {
		dir = "-" + dir
	}
	return fmt.Sprintf("new-%s-%s-%s", repo[len(repo)-1]+dir, did.ManifestID.Flavor, did.Cluster)

}
