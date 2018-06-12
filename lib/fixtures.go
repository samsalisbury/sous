package sous

import (
	"fmt"

	"github.com/samsalisbury/semv"
)

// These functions gin up fixtures for various complex structs to be used where
// a tested component needs e.g. a real manifest but where the manifest itself
// isn't important.
// These functions all follow the pattern
//    xxxFixture(name string) xxx {
//      switch name {
//        default:
//          [a vanilla instance]
//        ...
// If more specific forms are required, add them as "named" fixtures here.

var sequences = map[string]int{}

func seq(name string) int {
	n, has := sequences[name]
	if has {
		n = n + 1
		sequences[name] = n
		return n
	}
	sequences[name] = 0
	return 0
}

// DeployConfigFixture returns a fixtured DeployConfig. There are no defined stereotypes of this fixture yet.
func DeployConfigFixture(name string) DeployConfig {
	switch name {
	default:
		return DeployConfig{
			Resources: Resources{
				"cpus":   "0.100",
				"memory": "356",
				"ports":  "2",
			},
			Metadata:     Metadata{},
			Env:          Env{},
			NumInstances: 1,
			Volumes:      Volumes{},
			Startup: Startup{
				SkipCheck: true,
			},
		}
	}
}

// DeploymentFixture returns a fixtured Deployment. Stereotypes:
// * (default) a repeating "example" repo
// * sequenced-repo - uses sequences to return distinct deployments
func DeploymentFixture(name string) *Deployment {
	switch name {
	case "sequenced-repo":
		return &Deployment{
			DeployConfig: DeployConfigFixture(""),
			ClusterName:  "cluster-1",
			SourceID: SourceID{
				Location: SourceLocation{
					Repo: fmt.Sprintf("github.com/opentable/example-%d", seq("dep-repo")),
					Dir:  "",
				},
				Version: semv.MustParse("0.0.1"),
			},
			Flavor: "",
			Owners: OwnerSet{},
			Kind:   ManifestKindService,
			//Cluster *Cluster,
		}
	default:
		return &Deployment{
			DeployConfig: DeployConfigFixture(""),
			ClusterName:  "cluster-1",
			SourceID: SourceID{
				Location: SourceLocation{
					Repo: "github.com/opentable/example",
					Dir:  "",
				},
				Version: semv.MustParse("0.0.1"),
			},
			Flavor: "",
			Owners: OwnerSet{},
			Kind:   ManifestKindService,
			//Cluster *Cluster,
		}
	}
}

// DeployableFixture returns a fixtured Deployment. There are no defined stereotypes of this fixture yet.
func DeployableFixture(name string) *Deployable {
	switch name {
	default:
		return &Deployable{
			Status:     DeployStatusActive,
			Deployment: DeploymentFixture(""),
			BuildArtifact: &BuildArtifact{
				Type: "docker",
				//Name:      "dockerhub.io/example:0.0.1",
				DigestReference: "dockerhub.io/example@sha256:012345678901234567890123456789AB012345678901234567890123456789AB",
				Qualities:       []Quality{},
			},
		}
	}
}

// ManifestFixture returns a fixtured Manifest. Stereotypes are:
// * simple
// * with-metadata
func ManifestFixture(name string) *Manifest {
	switch name {
	default:
		panic("testManifest: unknown name: " + name)
	case "simple":
		return &Manifest{
			Source: SourceLocation{Repo: "github.com/opentable/project-one"},
			Owners: []string{"sam", "judson"},
			Kind:   ManifestKindService,
			Deployments: DeploySpecs{
				"ci": DeploySpec{
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus":   "0.1",
							"memory": "100",
							"ports":  "1",
						},
						Startup: Startup{
							CheckReadyURIPath: "certainly/i/am/healthy",
						},
					},
				},
			},
		}

	case "with-metadata":
		return &Manifest{
			Source: SourceLocation{Repo: "github.com/opentable/metadata-ish"},
			Owners: []string{"owner1"},
			Kind:   ManifestKindService,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("1.0.0"),
					DeployConfig: DeployConfig{
						Metadata: Metadata{
							"BuildBranch": "master",
							"DeployOn":    "build success",
						},
						NumInstances: 2,
					},
				},
				"cluster-2": {
					Version: semv.MustParse("2.0.0"),
					DeployConfig: DeployConfig{
						Metadata: Metadata{
							"BuildBranch": "master",
							"DeployOn":    "version advance",
						},
						NumInstances: 3,
					},
				},
			},
		}
	}
}
