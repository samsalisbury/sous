package test

import (
	"github.com/opentable/sous/sous"
	"github.com/samsalisbury/semv"
)

/*
func TestResolve(t *testing.T) {
	assert := assert.New(t)

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			"it": sous.Cluster{
				BaseURL: singularityURL,
			},
		},
	}

	stateOneTwo := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"one": manifest("https://github.com/opentable/one", "1.1.1"),
			"two": manifest("https://github.com/opentable/two", "1.1.1"),
		},
	}
	stateTwoThree := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"two":   manifest("https://github.com/opentable/two", "1.1.1"),
			"three": manifest("https://github.com/opentable/three", "1.1.1"),
		},
	}

	Resolve(stateOneTwo)
	// one and two are running
	Resolve(stateTwoThree)
	// two and three are running, not one

}
*/

func manifest(sourceURL, version string) sous.Manifest {
	return sous.Manifest{
		Source: sous.SourceLocation{
			RepoURL:    sous.RepoURL(sourceURL),
			RepoOffset: sous.RepoOffset(""),
		},
		Owners: []string{`xyz`},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			"it": sous.PartialDeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources:    sous.Resources{}, //map[string]string
					Args:         []string{},
					Env:          sous.Env{}, //map[s]s
					NumInstances: 1,
				},
				Version: semv.MustParse(version),
				//clusterName: "it",
			},
		},
	}
}
