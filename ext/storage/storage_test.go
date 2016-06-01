package storage

import (
	"os/exec"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/yaml"
	"github.com/samsalisbury/semv"
)

func TestWriteState(t *testing.T) {

	s := exampleState()

	if err := WriteState("test_output", s); err != nil {
		t.Fatal(err)
	}

	d := exec.Command("diff", "-r", "test_data/", "test_output/")
	out, err := d.CombinedOutput()
	if err != nil {
		t.Log("Output not as expected:")
		t.Log(string(out))
		t.Fatal()
	}
}

func TestReadState(t *testing.T) {

	actual, err := ReadState("test_data/")
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
		t.Log("Got >>>>>>>>>>>>>>>>>>>>>>")
		t.Logf("% +v", actualYAML)
		t.Log("Expected >>>>>>>>>>>>>>>>>")
		t.Logf("% +v", expectedYAML)
		t.Fatal()
	}
}

func exampleState() *sous.State {
	return &sous.State{
		Manifests: sous.Manifests{
			"github.com/opentable/sous": {
				Source: sous.SourceLocation{
					RepoURL: sous.RepoURL("github.com/opentable/sous"),
				},
				Owners: []string{"Judson", "Sam"},
				Kind:   "http-service",
				Deployments: map[string]sous.PartialDeploySpec{
					"Global": sous.PartialDeploySpec{
						DeployConfig: sous.DeployConfig{
							Resources: sous.Resources{
								"cpu": "0.1",
								"mem": "2GB",
							},
							NumInstances: 3,
						},
						Version: semv.MustParse("1.0.0"),
					},
					"cluster-1": sous.PartialDeploySpec{
						DeployConfig: sous.DeployConfig{
							Env: sous.Env{
								"SOME_DB_URL": "https://some.database",
							},
							NumInstances: 6,
						},
						Version: semv.MustParse("1.0.0-rc.1+deadbeef"),
					},
				},
			},
			"github.com/user/project": {
				Source: sous.SourceLocation{
					RepoURL: sous.RepoURL("github.com/user/project"),
				},
				Owners: []string{"Sous Team"},
				Kind:   "http-service",
				Deployments: map[string]sous.PartialDeploySpec{
					"other-cluster": {
						DeployConfig: sous.DeployConfig{
							Env: sous.Env{
								"DEBUG": "YES",
							},
						},
						Version: semv.MustParse("0.3.1-beta+b4d455ee"),
					},
				},
			},
		},
		Defs: sous.Defs{
			Clusters: sous.Clusters{
				"cluster-1": sous.Cluster{
					Kind:    "singularity",
					BaseURL: "http://singularity.example.com",
				},
				"other-cluster": sous.Cluster{
					Kind:    "singularity",
					BaseURL: "http://some.singularity.cluster",
				},
			},
			EnvVars:   sous.EnvDefs{},
			Resources: sous.ResDefs{},
		},
	}
}
