package otpl

import (
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/semv"
)

func TestManifestParser_ParseManifest(t *testing.T) {

	// Setup: write some files to disk.
	files := filemap.FileMap{
		"config/cluster1/singularity-request.json": `{
			"id": "request-1",
			"requestType": "SERVICE",
	        "owners": ["owner1@example.com"],
	        "instances": 2
	    }`,
		"config/cluster1/singularity.json": `{
		  "requestId": "request-1",
	      "resources": {
	          "cpus": 0.002,
	          "memoryMb": 96,
	          "numPorts": 1
	      },
	      "env": {
	          "SOME_VAR": "22"
	      }
	    }`,
		"config/cluster1.flavor1/singularity-request.json": `{
			"id": "request-2",
			"requestType": "SERVICE",
	        "owners": ["owner1@example.com"],
	        "instances": 2
	    }`,
		"config/cluster1.flavor1/singularity.json": `{
		  "requestId": "request-2",
	      "resources": {
	          "cpus": 0.002,
	          "memoryMb": 96,
	          "numPorts": 1
	      },
	      "env": {
	          "SOME_VAR": "22",
	          "OT_ENV_FLAVOR": "flavor1"
	      }
	    }`,
	}

	const testDataDir = "testdata/gen"

	var actual sous.Manifests

	if fileMapErr := files.Session(testDataDir, func() {
		wd, err := shell.DefaultInDir(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		ls, _ := logging.NewLogSinkSpy()
		actual = NewManifestParser(ls).ParseManifests(wd)
	}); fileMapErr != nil {
		t.Fatal(fileMapErr)
	}

	expected := sous.NewManifests(
		&sous.Manifest{
			Flavor: "",
			Owners: []string{"owner1@example.com"},
			Kind:   "",
			Deployments: sous.DeploySpecs{
				"cluster1": sous.DeploySpec{
					DeployConfig: sous.DeployConfig{
						Resources: sous.Resources{
							"cpus":   "0.002",
							"memory": "96",
							"ports":  "1",
						},
						Metadata: sous.Metadata(nil),
						Env: sous.Env{
							"SOME_VAR": "22",
						},
						NumInstances: 2,
						Volumes:      sous.Volumes(nil),
					},
					Version: semv.MustParse("0.0.0"),
				},
			},
		},
		&sous.Manifest{
			Flavor: "flavor1",
			Owners: []string{"owner1@example.com"},
			Kind:   "",
			Deployments: sous.DeploySpecs{
				"cluster1": sous.DeploySpec{
					DeployConfig: sous.DeployConfig{
						Resources: sous.Resources{
							"cpus":   "0.002",
							"memory": "96",
							"ports":  "1",
						},
						Metadata: sous.Metadata(nil),
						Env: sous.Env{
							"SOME_VAR":      "22",
							"OT_ENV_FLAVOR": "flavor1",
						},
						NumInstances: 2,
						Volumes:      sous.Volumes(nil),
					},
					Version: semv.MustParse("0.0.0"),
				},
			},
		},
	)

	if different, diffs := expected.Diff(actual); different {
		t.Errorf("parsed manifest not as expected")
		for _, diff := range diffs {
			t.Error(diff)
		}
	}
}
