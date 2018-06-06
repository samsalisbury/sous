package sous

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

var project1 = SourceLocation{Repo: "github.com/user/project"}
var project2 = SourceLocation{Repo: "github.com/user/scheduled"}
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

func makeTestDefs() Defs {
	return Defs{
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
		Resources: FieldDefinitions{
			{Name: "cpus", Type: "float"},
			{Name: "mem", Type: "memory_size"},
		},
	}
}

var omalleyReadyURI = "/ow/baby"

func makeTestManifests() Manifests {
	return NewManifests(
		&Manifest{
			Source: project1,
			Owners: []string{"owner1"},
			Kind:   ManifestKindService,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("1.0.0"),
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "1",
							"mem":  "1024",
						},
						Env: Env{
							"ALL":               "IS ONE",
							"ENV_1":             "ENV ONE",
							"CLUSTER_LONG_NAME": "Cluster One",
						},
						Metadata: Metadata{
							"everybody": "wants to be a cat",
							"name":      "O'Malley",
						},
						NumInstances: 2,
						Startup: Startup{
							CheckReadyURIPath: omalleyReadyURI,
						},
						SingularityRequestID: "project1-cluster-1",
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
							"ALL":               "IS ONE",
							"ENV_2":             "ENV TWO",
							"CLUSTER_LONG_NAME": "I Like To Call It 'Cluster Two'",
						},
						Metadata: Metadata{
							"everybody": "wants to be a cat",
							"name":      "Duchess",
						},
						NumInstances:         3,
						SingularityRequestID: "project1-cluster-2",
					},
				},
			},
		},
		&Manifest{
			Source: project1,
			Flavor: "some-flavor",
			Owners: []string{"owner1flav"},
			Kind:   ManifestKindService,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("1.0.1"),
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "1.5",
							"mem":  "1024",
						},
						Env: Env{
							"ENV_1": "ENV ONE FLAVORED",
						},
						NumInstances:         4,
						SingularityRequestID: "project1-some-flavor-cluster-1",
					},
				},
				"cluster-2": {
					Version: semv.MustParse("2.0.1"),
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "2.5",
							"mem":  "2048",
						},
						Env: Env{
							"ENV_2": "ENV TWO FLAVORED",
						},
						NumInstances:         5,
						SingularityRequestID: "project1-some-flavor-cluster-2",
					},
				},
			},
		},
		&Manifest{
			Source: project2,
			Kind:   ManifestKindScheduled,
			Deployments: DeploySpecs{
				"cluster-2": {
					Version: semv.MustParse("0.2.4"),
					DeployConfig: DeployConfig{
						Schedule: "* */2 * * *",
						Resources: Resources{
							"cpus": "0.4",
							"mem":  "256",
						},
						SingularityRequestID: "project2-cluster-2",
					},
				},
			},
		},
	)
}

func makeTestDeployments() (Deployments, error) {
	return makeTestManifests().Deployments(makeTestDefs())
}

var expectedDeployments = NewDeployments(
	&Deployment{
		SourceID:    project1.SourceID(semv.MustParse("1.0.0")),
		ClusterName: "cluster-1",
		Cluster:     cluster1,
		Kind:        ManifestKindService,
		Owners:      NewOwnerSet("owner1"),
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "1",
				"mem":  "1024",
			},
			Env: Env{
				"ALL":               "IS ONE",
				"ENV_1":             "ENV ONE",
				"CLUSTER_LONG_NAME": "Cluster One",
			},
			Metadata: Metadata{
				"everybody": "wants to be a cat",
				"name":      "O'Malley",
			},
			NumInstances: 2,
			Startup: Startup{
				CheckReadyURIPath: omalleyReadyURI,
			},
			SingularityRequestID: "project1-cluster-1",
		},
	},
	&Deployment{
		SourceID:    project1.SourceID(semv.MustParse("2.0.0")),
		ClusterName: "cluster-2",
		Cluster:     cluster2,
		Kind:        ManifestKindService,
		Owners:      NewOwnerSet("owner1"),
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "2",
				"mem":  "2048",
			},
			Env: Env{
				"ALL":   "IS ONE",
				"ENV_2": "ENV TWO",
				//"CLUSTER_LONG_NAME": "Cluster Two",
				"CLUSTER_LONG_NAME": "I Like To Call It 'Cluster Two'",
			},
			Metadata: Metadata{
				"everybody": "wants to be a cat",
				"name":      "Duchess",
			},
			NumInstances:         3,
			SingularityRequestID: "project1-cluster-2",
		},
	},
	&Deployment{
		Flavor:      "some-flavor",
		SourceID:    project1.SourceID(semv.MustParse("1.0.1")),
		ClusterName: "cluster-1",
		Cluster:     cluster1,
		Kind:        ManifestKindService,
		Owners:      NewOwnerSet("owner1flav"),
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "1.5",
				"mem":  "1024",
			},
			Env: Env{
				"ENV_1":             "ENV ONE FLAVORED",
				"CLUSTER_LONG_NAME": "Cluster One",
			},
			NumInstances:         4,
			SingularityRequestID: "project1-some-flavor-cluster-1",
		},
	},
	&Deployment{
		Flavor:      "some-flavor",
		SourceID:    project1.SourceID(semv.MustParse("2.0.1")),
		ClusterName: "cluster-2",
		Cluster:     cluster2,
		Kind:        ManifestKindService,
		Owners:      NewOwnerSet("owner1flav"),
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "2.5",
				"mem":  "2048",
			},
			Env: Env{
				"ENV_2":             "ENV TWO FLAVORED",
				"CLUSTER_LONG_NAME": "Cluster Two",
			},
			NumInstances:         5,
			SingularityRequestID: "project1-some-flavor-cluster-2",
		},
	},
	&Deployment{
		SourceID:    project2.SourceID(semv.MustParse("0.2.4")),
		ClusterName: "cluster-2",
		Cluster:     cluster2,
		Kind:        ManifestKindScheduled,
		DeployConfig: DeployConfig{
			Schedule: "* */2 * * *",
			Resources: Resources{
				"cpus": "0.4",
				"mem":  "256",
			},
			Env: Env{
				"CLUSTER_LONG_NAME": "Cluster Two",
			},
			SingularityRequestID: "project2-cluster-2",
		},
	},
)

func TestState_DeploymentsCloned(t *testing.T) {
	actualDeployments, err := makeTestDeployments()
	if err != nil {
		t.Fatal(err)
	}
	actualDeployments = actualDeployments.Clone()

	exSnap := expectedDeployments.Snapshot()
	if len(actualDeployments.Snapshot()) != len(exSnap) {
		t.Errorf("deployments different lengths: expected %d, got %d", len(exSnap), len(actualDeployments.Snapshot()))
	}
	for id, expected := range exSnap {
		actual, ok := actualDeployments.Get(id)
		if !ok {
			t.Errorf("missing deployment %q", id)
			continue
		}
		if !actual.Equal(expected) {
			t.Errorf("%q\n\ngot:\n%v\n\nwant:\n%v\n", id, jsonDump(actual), jsonDump(expected))
		}
	}
}

var TestStateIndependentDeploySpecsState Manifests

func TestState_IndependentDeploySpecs(t *testing.T) {
	originalManifests := makeTestManifests()
	defs := makeTestDefs()
	originalDeployments, err := originalManifests.Deployments(defs)
	if err != nil {
		t.Fatal(err)
	}
	mid := ManifestID{Source: SourceLocation{Repo: "github.com/user/project"}}
	did := DeploymentID{ManifestID: mid, Cluster: "cluster-1"}
	originalDeployment, ok := originalDeployments.Get(did)
	if !ok {
		t.Fatalf("deployment %v not found", did)
	}
	originalJSON := jsonDump(originalDeployment)
	ls, _ := logging.NewLogSinkSpy()
	// We don't care about this result, but write it to global state to avoid
	// any compiler optimisation from eliding the call.
	TestStateIndependentDeploySpecsState, err = originalDeployments.PutbackManifests(defs, originalManifests, ls)
	if err != nil {
		t.Errorf("Expected no error while updating manifests, got: %v", err)
	}
	newDeployment, ok := originalDeployments.Get(did)
	if !ok {
		t.Fatalf("deployment %v went missing", did)
	}
	newJSON := jsonDump(newDeployment)

	if originalJSON != newJSON {
		t.Fatalf("original deployspec changed:\n\noriginal:\n%s\n\nnow:\n%s", originalJSON, newJSON)
	}
}

func TestState_Deployments(t *testing.T) {
	actualDeployments, err := makeTestDeployments()
	if err != nil {
		t.Fatal(err)
	}
	compareDeployments(t, expectedDeployments, actualDeployments)
}

func TestState_DeploymentsBounce(t *testing.T) {
	defs := makeTestDefs()
	ls, _ := logging.NewLogSinkSpy()
	bounceManifests, err := expectedDeployments.Clone().PutbackManifests(defs, makeTestManifests(), ls)
	if err != nil {
		t.Fatal(err)
	}
	actualDeployments, err := bounceManifests.Deployments(defs)
	if err != nil {
		t.Fatal(err)
	}
	compareDeployments(t, expectedDeployments, actualDeployments)
}

func TestDeployments_Manifests(t *testing.T) {
	defs := makeTestDefs()

	ls, _ := logging.NewLogSinkSpy()
	actualManifests, err := expectedDeployments.Clone().PutbackManifests(defs, makeTestManifests(), ls)
	if err != nil {
		t.Fatal(err)
	}
	expectedManifests := makeTestManifests()

	compareManifests(t, expectedManifests, actualManifests)
}

func TestDeployments_ManifestsBounce(t *testing.T) {
	defs := makeTestDefs()

	bounced := makeTestManifests()

	bounceDeployments, err := bounced.Deployments(makeTestDefs())
	if err != nil {
		t.Fatal(err)
	}

	ls, _ := logging.NewLogSinkSpy()
	actualManifests, err := bounceDeployments.PutbackManifests(defs, bounced, ls)
	if err != nil {
		t.Fatal(err)
	}
	expectedManifests := makeTestManifests()

	compareManifests(t, expectedManifests, actualManifests)
}

func TestDeployments_PutbackManifestMissingManifestElides(t *testing.T) {
	defs := makeTestDefs()
	deps := NewDeployments(
		&Deployment{
			SourceID: project1.SourceID(semv.MustParse("2.0.0")),

			Cluster:     cluster1,
			ClusterName: cluster1.Name,
			DeployConfig: DeployConfig{
				Env: Env{
					"CLUSTER_LONG_NAME": "Cluster One",
					"PRESENT":           "here",
				},
			},
		},
	)

	ls, _ := logging.NewLogSinkSpy()
	zeroMs := NewManifests()
	ms, err := deps.PutbackManifests(defs, zeroMs, ls)
	assert.NoError(t, err)
	m, yes := ms.Any(func(*Manifest) bool { return true })
	assert.True(t, yes)
	assert.NotContains(t, m.Deployments["cluster-1"].Env, "CLUSTER_LONG_NAME")
	assert.Contains(t, m.Deployments["cluster-1"].Env, "PRESENT")
}

func TestDeployments_PutbackManifestOriginalLacksClusterElides(t *testing.T) {
	defs := makeTestDefs()
	deps := NewDeployments(
		&Deployment{
			SourceID: project1.SourceID(semv.MustParse("2.0.0")),

			Cluster:     cluster1,
			ClusterName: cluster1.Name,
			DeployConfig: DeployConfig{
				Env: Env{
					"CLUSTER_LONG_NAME": "Cluster One",
					"PRESENT":           "here",
				},
			},
		},
	)

	skewMs := NewManifests(
		&Manifest{
			Source: project1,
			Deployments: DeploySpecs{
				"cluster-2": { // != cluster-1
					Version: semv.MustParse("2.0.0"),
					DeployConfig: DeployConfig{
						Env: Env{
							"CLUSTER_LONG_NAME": "Cluster One",
							"PRESENT":           "here",
						},
					},
				},
			},
		},
	)
	ls, _ := logging.NewLogSinkSpy()
	ms, err := deps.PutbackManifests(defs, skewMs, ls)
	assert.NoError(t, err)
	m, yes := ms.Any(func(*Manifest) bool { return true })
	assert.True(t, yes)
	assert.NotContains(t, m.Deployments["cluster-1"].Env, "CLUSTER_LONG_NAME")
	assert.Contains(t, m.Deployments["cluster-1"].Env, "PRESENT")
}

func TestDeployments_PutbackManifestOriginalLacksEnvElides(t *testing.T) {
	defs := makeTestDefs()
	deps := NewDeployments(
		&Deployment{
			SourceID: project1.SourceID(semv.MustParse("2.0.0")),

			Cluster:     cluster1,
			ClusterName: cluster1.Name,
			DeployConfig: DeployConfig{
				Env: Env{
					"CLUSTER_LONG_NAME": "Cluster One",
					"PRESENT":           "here",
				},
			},
		},
	)

	skewMs := NewManifests(
		&Manifest{
			Source: project1,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("2.0.0"),
					DeployConfig: DeployConfig{
						Env: Env{
							//"CLUSTER_LONG_NAME": "Cluster One",
							"PRESENT": "here",
						},
					},
				},
			},
		},
	)

	ls, _ := logging.NewLogSinkSpy()
	ms, err := deps.PutbackManifests(defs, skewMs, ls)
	assert.NoError(t, err)
	m, yes := ms.Any(func(*Manifest) bool { return true })
	assert.True(t, yes)
	assert.NotContains(t, m.Deployments["cluster-1"].Env, "CLUSTER_LONG_NAME")
	assert.Contains(t, m.Deployments["cluster-1"].Env, "PRESENT")
}

func TestDeployments_PutbackManifestHasEnvRetains(t *testing.T) {
	defs := makeTestDefs()
	deps := NewDeployments(
		&Deployment{
			SourceID: project1.SourceID(semv.MustParse("2.0.0")),

			Cluster:     cluster1,
			ClusterName: cluster1.Name,
			DeployConfig: DeployConfig{
				Env: Env{
					"CLUSTER_LONG_NAME": "Cluster One",
					"PRESENT":           "here",
				},
			},
		},
	)

	skewMs := NewManifests(
		&Manifest{
			Source: project1,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("2.0.0"),
					DeployConfig: DeployConfig{
						Env: Env{
							"CLUSTER_LONG_NAME": "Cluster One",
							"PRESENT":           "here",
						},
					},
				},
			},
		},
	)

	ls, _ := logging.NewLogSinkSpy()
	ms, err := deps.PutbackManifests(defs, skewMs, ls)
	assert.NoError(t, err)
	m, yes := ms.Any(func(*Manifest) bool { return true })
	assert.True(t, yes)
	assert.Contains(t, m.Deployments["cluster-1"].Env, "CLUSTER_LONG_NAME")
	assert.Contains(t, m.Deployments["cluster-1"].Env, "PRESENT")
}

func compareDeployments(t *testing.T, expectedDeployments, actualDeployments Deployments) {
	exSnap := expectedDeployments.Snapshot()
	if len(actualDeployments.Snapshot()) != len(exSnap) {
		t.Errorf("deployments different lengths, expected %d got %d", len(exSnap), len(actualDeployments.Snapshot()))
	}
	for id, expected := range exSnap {
		actual, ok := actualDeployments.Get(id)
		if !ok {
			t.Errorf("missing deployment %q", id)
			continue
		}

		// XXX uses deployment.Diff
		if different, diffs := actual.Diff(expected); different {
			t.Errorf("\n\ngot:\n%v\ndifferences:\n%s\n", jsonDump(actual), strings.Join(diffs, "\n"))
		}
	}
}

func compareManifests(t *testing.T, expectedManifests, actualManifests Manifests) {
	actualLen := actualManifests.Len()
	expectedLen := expectedManifests.Len()
	if actualLen != expectedLen {
		t.Fatalf("got %d manifests; want %d", actualLen, expectedLen)
	}
	for _, mid := range expectedManifests.Keys() {
		expected, ok := expectedManifests.Get(mid)
		if !ok {
			t.Errorf("missing expected manifest %q", mid)
			continue
		}
		actual, ok := actualManifests.Get(mid)
		if !ok {
			t.Errorf("missing manifest %q", mid)
			continue
		}
		different, differences := actual.Diff(expected)
		if different {
			t.Errorf("manifests not as expected: \n  %s", strings.Join(differences, "\n  "))
			continue
		}
		// Check all expected DeploySpecs are in actual.
		for clusterName := range expected.Deployments {
			did := DeploymentID{Cluster: clusterName, ManifestID: expected.ID()}
			_, ok := actual.Deployments[clusterName]
			if !ok {
				t.Errorf("deployment %q missing", did)
			}
		}
		// Check actual contains only the expected DeploySpecs.
		for clusterName := range actual.Deployments {
			did := DeploymentID{Cluster: clusterName, ManifestID: actual.ID()}
			_, ok := expected.Deployments[clusterName]
			if !ok {
				t.Errorf("extra deployment %q", did)
			}
		}
	}
}

func jsonDump(v interface{}) string { b, _ := json.MarshalIndent(v, "", "  "); return string(b) }
