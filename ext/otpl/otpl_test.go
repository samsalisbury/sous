package otpl

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/semv"
)

func TestParseSingularityJSON_ok(t *testing.T) {

	in := `
	{
		"resources": {
			"numPorts": 1,
			"memoryMb": 1,
			"cpus": 1
		},
		"env": {
			"ENV_1": "val 1"
		}
	}`

	want := SingularityJSON{
		Resources: SingularityResources{
			"numPorts": 1,
			"memoryMb": 1,
			"cpus":     1,
		},
		Env: sous.Env{
			"ENV_1": "val 1",
		},
	}

	got, err := parseSingularityJSON(in)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got -->\n% #v\nwant -->\n% #v", got, want)
	}
}

func TestParseSingularityJSON_err_fields(t *testing.T) {

	cases := []string{
		`{"invalid": {}}`,
		`{"env": {"ENV_1": "val 1"}, "invalid": {}}`,
		`
		{
			"resources": {
				"numPorts": 1,
				"memoryMb": 1,
				"cpus": 1
			},
			"env": {
				"ENV_1": "val 1"
			},
			"blahBlahInvalid": "hello"
		}`,
	}

	const wantPrefix = `unrecognised fields:`

	for i, in := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, gotErr := parseSingularityJSON(in)
			if gotErr == nil {
				t.Fatalf("got nil error; want error beginning %q", wantPrefix)
			}
			got := gotErr.Error()
			if !strings.HasPrefix(got, wantPrefix) {
				t.Errorf("got %q; want string with prefix %q", got, wantPrefix)
			}
		})
	}
}

func TestParseSingularityRequestJSON_ok(t *testing.T) {

	in := `
	{
		"instances": 1,
		"owners": ["owner1@example.com"]
	}`

	want := SingularityRequestJSON{
		Instances: 1,
		Owners:    []string{"owner1@example.com"},
	}

	got, err := parseSingularityRequestJSON(in)

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got -->\n% #v\nwant -->\n% #v", got, want)
	}
}

func TestParseSingularityRequestJSON_err_fields(t *testing.T) {

	cases := []string{
		`{"invalid": {}}`,
		`{"env": {"ENV_1": "val 1"}, "invalid": {}}`,
		`
		{
			"instances": 1,
			"owners": ["owner1@example.com"],
			"blahBlahInvalid": "hello"
		}`,
	}

	const wantPrefix = `unrecognised fields:`

	for i, in := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, gotErr := parseSingularityRequestJSON(in)
			if gotErr == nil {
				t.Fatalf("got nil error; want error beginning %q", wantPrefix)
			}
			got := gotErr.Error()
			if !strings.HasPrefix(got, wantPrefix) {
				t.Errorf("got %q; want string with prefix %q", got, wantPrefix)
			}
		})
	}
}

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

		ls, _ := logging.NewLogSinkSpy()
		// Shebang...
		actual = NewManifestParser(ls).ParseManifests(wd)
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
