package otpl

import (
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/semv"
)

func TestSingularityResources_SousResources(t *testing.T) {
	tests := []struct {
		Singularity SingularityResources
		Sous        sous.Resources
	}{
		{ // This won't really happen.
			SingularityResources{
				"cpu":    1,
				"ports":  1,
				"memory": 1,
			},
			sous.Resources{
				"cpu":    "1",
				"ports":  "1",
				"memory": "1",
			},
		},
		{ // Mapping singularity resource names to Sous ones.
			SingularityResources{
				"cpu":      1,
				"numPorts": 1,
				"memoryMb": 1,
			},
			sous.Resources{
				"cpu":    "1",
				"ports":  "1",
				"memory": "1",
			},
		},
	}

	for i, test := range tests {
		input := test.Singularity
		expected := test.Sous

		actual := input.SousResources()
		if !actual.Equal(expected) {
			t.Errorf("got resources %# v; want %# v; for input %d %# v",
				actual, expected, i, input)
		}
	}
}

func TestManifestParser_ParseManifest(t *testing.T) {

	// Setup: write some files to disk.
	files := filemap.FileMap{
		"config/cluster1/singularity-request.json": `{
	        "owners": ["owner1@example.com"],
	        "instances": 2,
	        "other fields": "are ignored"
	    }`,
		"config/cluster1/singularity.json": `{
	      "resources": {
	          "cpus": 0.002,
	          "memoryMb": 96,
	          "numPorts": 1
	      },
	      "env": {
	          "SOME_VAR": "22"
	      },
	      "other fields": "are ignored"
	    }`,
		"config/cluster1.flavor1/singularity-request.json": `{
	        "owners": ["owner1@example.com"],
	        "instances": 2,
	        "other fields": "are ignored"
	    }`,
		"config/cluster1.flavor1/singularity.json": `{
	      "resources": {
	          "cpus": 0.002,
	          "memoryMb": 96,
	          "numPorts": 1
	      },
	      "env": {
	          "SOME_VAR": "22",
	          "OT_ENV_FLAVOR": "flavor1"
	      },
	      "other fields": "are ignored"
	    }`,
	}

	const testDataDir = "testdata/gen"

	var actual sous.Manifests

	if fileMapErr := files.Session(testDataDir, func() {
		wd, err := shell.DefaultInDir(testDataDir)
		if err != nil {
			t.Fatal(err)
		}
		// Shebang...
		actual = NewManifestParser().ParseManifests(wd)
	}); fileMapErr != nil {
		t.Fatal(fileMapErr)
	}

	expected := sous.NewManifests(
		&sous.Manifest{
			//Source: sous.MustParseSourceLocation("github.com/test/project"),
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
			//Source: sous.MustParseSourceLocation("github.com/test/project"),
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
							"SOME_VAR": "22",
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
