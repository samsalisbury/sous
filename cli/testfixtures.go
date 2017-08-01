package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
)

func testManifest(name string) *sous.Manifest {
	switch name {
	default:
		panic("testManifest: unknown name: " + name)
	case "simple":
		return &sous.Manifest{
			Source: sous.SourceLocation{Repo: "github.com/opentable/project-one"},
			Owners: []string{"sam", "judson"},
			Kind:   sous.ManifestKindService,
			Deployments: sous.DeploySpecs{
				"ci": sous.DeploySpec{
					DeployConfig: sous.DeployConfig{
						Resources: sous.Resources{
							"cpus":   "0.1",
							"memory": "100",
							"ports":  "1",
						},
						Startup: sous.Startup{
							CheckReadyURIPath: "certainly/i/am/healthy",
						},
					},
				},
			},
		}

	case "with-metadata":
		return &sous.Manifest{
			Source: sous.SourceLocation{Repo: "github.com/opentable/metadata-ish"},
			Owners: []string{"owner1"},
			Kind:   sous.ManifestKindService,
			Deployments: sous.DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("1.0.0"),
					DeployConfig: sous.DeployConfig{
						Metadata: sous.Metadata{
							"BuildBranch": "master",
							"DeployOn":    "build success",
						},
						NumInstances: 2,
					},
				},
				"cluster-2": {
					Version: semv.MustParse("2.0.0"),
					DeployConfig: sous.DeployConfig{
						Metadata: sous.Metadata{
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
