package sous

import "testing"

func makeTestState() *State {
	project1 := SourceLocation{RepoURL: "github.com/user/project"}
	return &State{
		Defs: Defs{
			DockerRepo: "some.docker.repo",
			Clusters: Clusters{
				"cluster-1": {
					Name:    "cluster-1",
					Kind:    "singularity",
					BaseURL: "http://nothing.here.one",
					Env: EnvDefaults{
						"CLUSTER_LONG_NAME": Var("Cluster One"),
					},
				},
				"cluster-2": {
					Name:    "cluster-2",
					Kind:    "singularity",
					BaseURL: "http://nothing.here.two",
					Env: EnvDefaults{
						"CLUSTER_LONG_NAME": Var("Cluster Two"),
					},
				},
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
			},
		),
	}
}

func TestState_Deployments(t *testing.T) {
	_ = makeTestState()
}
