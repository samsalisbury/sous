package sous

import "github.com/samsalisbury/semv"

func deployConfigFixture(name string) DeployConfig {
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

func deploymentFixture(name string) *Deployment {
	switch name {
	default:
		return &Deployment{
			DeployConfig: deployConfigFixture(""),
			ClusterName:  "test-cluster",
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

func deployableFixture(name string) *Deployable {
	switch name {
	default:
		return &Deployable{
			Status:     DeployStatusActive,
			Deployment: deploymentFixture(""),
			BuildArtifact: &BuildArtifact{
				Type:      "docker",
				Name:      "dockerhub.io/example:0.0.1",
				Qualities: []Quality{},
			},
		}
	}
}

func manifestFixture(name string) *Manifest {
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
