package otpl

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
	"github.com/pkg/errors"
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
	files := fileMap{
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
	}
	if err := files.Write("testdata/gen"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := files.Delete("testdata/gen"); err != nil {
			t.Fatalf("cleanup failed: %s", err)
		}
	}()

	wd, err := shell.DefaultInDir("testdata/gen")
	if err != nil {
		t.Fatal(err)
	}

	actual := NewManifestParser().ParseManifest(wd)

	expected := &sous.Manifest{
		//Source: sous.MustParseSourceLocation("github.com/test/project"),
		Flavor: "",
		Owners: nil,
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
					Args:     []string(nil),
					Env: sous.Env{
						"SOME_VAR": "22",
					},
					NumInstances: 2,
					Volumes:      sous.Volumes(nil),
				},
				Version: semv.MustParse("0.0.0"),
			},
		},
	}

	if different, diffs := actual.Diff(expected); different {
		t.Errorf("parsed manifest not as expected")
		for _, diff := range diffs {
			t.Error(diff)
		}
	}
}

type fileMap map[string]string

func (f fileMap) Delete(dir string) error {
	return os.RemoveAll(dir)
}

func (f fileMap) Write(dir string) error {
	for name, contents := range f {
		name := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(name), 0777); err != nil {
			return err
		}
		if err := ioutil.WriteFile(name, []byte(contents), 0777); err != nil {
			if deleteErr := f.Delete(dir); err != nil {
				return errors.Wrapf(err, "error cleaning up: %s", deleteErr)
			}
			return errors.Wrapf(err, "error writing files, cleanup successful")
		}
	}
	return nil
}
