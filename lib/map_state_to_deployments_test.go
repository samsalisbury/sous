package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
)

var project1 = SourceLocation{Repo: "github.com/user/project"}
var cluster1 = &Cluster{
	Name:    "cluster-1",
	Kind:    "singularity",
	BaseURL: "http://nothing.here.one",
	Env: EnvDefaults{
		"CLUSTER_LONG_NAME": Var("Cluster One"),
	},
}
var cluster2 = &Cluster{
	Name:    "cluster-2",
	Kind:    "singularity",
	BaseURL: "http://nothing.here.two",
	Env: EnvDefaults{
		"CLUSTER_LONG_NAME": Var("Cluster Two"),
	},
}

func makeTestState() *State {
	return &State{
		Defs: Defs{
			DockerRepo: "some.docker.repo",
			Clusters: Clusters{
				"cluster-1": cluster1,
				"cluster-2": cluster2,
			},
			EnvVars: EnvDefs{
				{
					Name:  "CLUSTER_LONG_NAME",
					Desc:  "The human-friendly name of this cluster.",
					Scope: "cluster",
					Type:  VarType("string"),
				},
			},
			Resources: ResDefs{
				{Name: "cpus", Type: "float"},
				{Name: "mem", Type: "memory_size"},
			},
		},
		Manifests: NewManifests(
			&Manifest{
				Source: project1,
				Deployments: DeploySpecs{
					"cluster-1": {
						Version: semv.MustParse("1.0.0"),
						DeployConfig: DeployConfig{
							Resources: Resources{
								"cpus": "1",
								"mem":  "1024",
							},
							Env: Env{
								"ENV_1": "ENV ONE",
							},
							NumInstances: 2,
						},
					},
					"cluster-2": {
						Version: semv.MustParse("2.0.0"),
						DeployConfig: DeployConfig{
							Resources: Resources{
								"cpus": "2",
								"mem":  "2048",
							},
							Env: Env{
								"ENV_2": "ENV TWO",
							},
							NumInstances: 3,
						},
					},
				},
			},
		),
	}
}

var expectedDeployments = NewDeployments(
	&Deployment{
		SourceID:    project1.SourceID(semv.MustParse("1.0.0")),
		ClusterName: "cluster-1",
		Cluster:     cluster1,
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "1",
				"mem":  "1024",
			},
			Env: Env{
				"ENV_1":             "ENV ONE",
				"CLUSTER_LONG_NAME": "Cluster One",
			},
			NumInstances: 2,
		},
	},
	&Deployment{
		SourceID:    project1.SourceID(semv.MustParse("2.0.0")),
		ClusterName: "cluster-2",
		Cluster:     cluster2,
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "2",
				"mem":  "2048",
			},
			Env: Env{
				"ENV_2":             "ENV TWO",
				"CLUSTER_LONG_NAME": "Cluster Two",
			},
			NumInstances: 3,
		},
	},
)

func TestState_Deployments(t *testing.T) {
	t.Skipf("until cluster/clusternickname sorted")
	actualDeployments, err := makeTestState().Deployments()
	if err != nil {
		t.Fatal(err)
	}
	for id, expected := range expectedDeployments.Snapshot() {
		actual, ok := actualDeployments.Get(id)
		if !ok {
			t.Errorf("missing deployment %q", id)
			continue
		}
		if actual != expected {
			t.Errorf("got %v; want %v", actual, expected)
		}
	}
}
