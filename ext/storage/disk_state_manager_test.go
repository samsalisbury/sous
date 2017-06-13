package storage

import (
	"os"
	"os/exec"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

func TestWriteState(t *testing.T) {

	s := exampleState()

	if err := os.RemoveAll("testdata/out"); err != nil {
		t.Fatal(err)
	}

	dsm := NewDiskStateManager("testdata/out")

	if err := dsm.WriteState(s, sous.StateContext{}); err != nil {
		t.Fatal(err)
	}

	d := exec.Command("diff", "-r", "testdata/in", "testdata/out")
	out, err := d.CombinedOutput()
	if err != nil {
		t.Log("Output not as expected:")
		t.Log(string(out))
		t.Fatal("")
	}
}

func TestReadState(t *testing.T) {

	dsm := NewDiskStateManager("testdata/in")

	actual, err := dsm.ReadState(sous.StateContext{})
	if err != nil {
		t.Fatal(err)
	}

	expected := exampleState()

	actualYAML, err := yaml.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}

	expectedYAML, err := yaml.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}

	if string(actualYAML) != string(expectedYAML) {
		t.Log("Expected >>>>>>>>>>>>>>>>>")
		t.Logf("\n% +v", string(expectedYAML))
		t.Log("Got >>>>>>>>>>>>>>>>>>>>>>")
		t.Logf("\n% +v", string(actualYAML))
		t.Fatal("")
	}
}

func TestReadState_empty(t *testing.T) {
	dsm := NewDiskStateManager("testdata/nonexistent")
	actual, err := dsm.ReadState(sous.StateContext{})
	if err != nil && !os.IsNotExist(errors.Cause(err)) {
		t.Fatal(err)
	}
	d, err := actual.Deployments()
	if err != nil {
		t.Fatal(err)
	}
	if d.Len() != 0 {
		t.Errorf("got len %d; want %d", d.Len(), 0)
	}
}

func exampleState() *sous.State {
	sl := sous.SourceLocation{
		Repo: "github.com/opentable/sous",
	}
	sl2 := sous.SourceLocation{
		Repo: "github.com/user/project",
	}
	return &sous.State{
		Manifests: sous.NewManifests(
			&sous.Manifest{
				Source: sl,
				Owners: []string{"Judson", "Sam"},
				Kind:   "http-service",
				Deployments: map[string]sous.DeploySpec{
					"cluster-1": sous.DeploySpec{
						DeployConfig: sous.DeployConfig{
							Env: sous.Env{
								"SOME_DB_URL": "https://some.database",
							},
							Resources: sous.Resources{
								"cpus":   "0.1",
								"memory": "2GB",
								"ports":  "1",
							},
							NumInstances: 6,
							Volumes:      sous.Volumes{},
						},
						Version: semv.MustParse("1.0.0-rc.1+deadbeef"),
					},
				},
			},
			&sous.Manifest{
				Source: sl2,
				Owners: []string{"Sous Team"},
				Kind:   "http-service",
				Deployments: map[string]sous.DeploySpec{
					"other-cluster": {
						DeployConfig: sous.DeployConfig{
							Env: sous.Env{
								"DEBUG": "YES",
							},
							Resources: sous.Resources{
								"cpus":   "1",
								"memory": "256MB",
								"ports":  "1",
							},
							Volumes: sous.Volumes{},
						},
						Version: semv.MustParse("0.3.1-beta+b4d455ee"),
					},
				},
			},
		),
		Defs: sous.Defs{
			DockerRepo: "docker.somewhere.horse",
			Clusters: sous.Clusters{
				"cluster-1": &sous.Cluster{
					Kind:    "singularity",
					BaseURL: "http://singularity.example.com",
				},
				"other-cluster": &sous.Cluster{
					Kind:    "singularity",
					BaseURL: "http://some.singularity.cluster",
				},
			},
			EnvVars:   sous.EnvDefs{},
			Resources: sous.FieldDefinitions{},
			Metadata:  sous.FieldDefinitions{},
		},
	}
}
