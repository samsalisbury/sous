package sous

import (
	"encoding/json"
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
				Owners: []string{"owner1"},
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
		Owners:      OwnerSet{"owner1": struct{}{}},
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
		Owners:      OwnerSet{"owner1": struct{}{}},
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
		if !actual.Equal(expected) {
			t.Errorf("\n\ngot:\n%v\n\nwant:\n%v\n", jsonDump(actual), jsonDump(expected))
		}
	}
}

func TestDeployments_Manifests(t *testing.T) {
	defs := makeTestState().Defs

	actualManifests, err := expectedDeployments.Manifests(defs)
	if err != nil {
		t.Fatal(err)
	}
	expectedManifests := makeTestState().Manifests
	actualLen := actualManifests.Len()
	expectedLen := expectedManifests.Len()
	if actualLen != expectedLen {
		t.Fatalf("got %d manifests; want %d", actualLen, expectedLen)
	}
	for _, sl := range expectedManifests.Keys() {
		t.Log(sl, "READING")
		expected, _ := expectedManifests.Get(sl)
		actual, ok := actualManifests.Get(sl)
		if !ok {
			t.Errorf("missing manifest %q", sl)
			continue
		}
		if !actual.Equal(expected) {
			t.Errorf("\n\ngot:\n%v\n\nwant:\n%v\n", jsonDump(actual), jsonDump(expected))
		}
		// Check all expected DeploySpecs are in actual.
		for clusterName := range expected.Deployments {
			did := DeployID{Cluster: clusterName, ManifestID: expected.ID()}
			_, ok := actual.Deployments[clusterName]
			if !ok {
				t.Errorf("deployment %q missing", did)
			} else {
				t.Logf("GOT DEPLOYMENT %q", did)
			}
		}
		// Check actual contains only the expected DeploySpecs.
		for clusterName := range actual.Deployments {
			did := DeployID{Cluster: clusterName, ManifestID: actual.ID()}
			_, ok := expected.Deployments[clusterName]
			if !ok {
				t.Errorf("extra deployment %q", did)
			}
		}
		t.Log(sl, "OK")
	}
}

func jsonDump(v interface{}) string { b, _ := json.MarshalIndent(v, "", "  "); return string(b) }
