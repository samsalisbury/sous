package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
)

var detectTests = []struct {
	Dockerfile   string
	DetectResult *sous.DetectResult
	Error        string
}{
	{
		Error: "Dockerfile does not exist",
	},
	{
		Dockerfile: `FROM blah`,
		DetectResult: &sous.DetectResult{
			Compatible: true,
			Data:       detectData{},
		},
	},
	{
		Dockerfile: `FROM blah
ARG APP_VERSION`,
		DetectResult: &sous.DetectResult{
			Compatible: true,
			Data: detectData{
				HasAppVersionArg: true,
			},
		},
	},
	{
		Dockerfile: `FROM blah
ARG APP_REVISION`,
		DetectResult: &sous.DetectResult{
			Compatible: true,
			Data: detectData{
				HasAppRevisionArg: true,
			},
		},
	},
	{
		Dockerfile: `FROM blah
ARG APP_VERSION
ARG APP_REVISION`,
		DetectResult: &sous.DetectResult{
			Compatible: true,
			Data: detectData{
				HasAppVersionArg:  true,
				HasAppRevisionArg: true,
			},
		},
	},
	{
		Dockerfile: `FROM blah
ARG APP_VERSION="1.0.0"
ARG APP_REVISION="cabba9e"
		`,
		DetectResult: &sous.DetectResult{
			Compatible: true,
			Data: detectData{
				HasAppVersionArg:  true,
				HasAppRevisionArg: true,
			},
		},
	},
}

func TestDetect(t *testing.T) {
	const baseDir = "testdata/gen"
	os.RemoveAll(baseDir)
	for i, test := range detectTests {
		testDir := fmt.Sprintf("%s/%d", baseDir, i)
		if err := os.MkdirAll(testDir, 0777); err != nil {
			t.Fatal(err)
		}
		sh, err := shell.DefaultInDir(testDir)
		if err != nil {
			t.Fatal(err)
		}
		c := &sous.BuildContext{
			Sh:     sh,
			Source: sous.SourceContext{},
		}
		if test.Dockerfile != "" {
			dockerfilePath := path.Join(testDir, "Dockerfile")
			dockerfileBytes := []byte(test.Dockerfile)
			if err := ioutil.WriteFile(dockerfilePath, dockerfileBytes, 0777); err != nil {
				t.Fatal(err)
			}
		}
		dr, err := (&DockerfileBuildpack{}).Detect(c)
		if err := assertError(test.Error, err); err != nil {
			t.Error(err)
		}
		if err := assertResult(test.DetectResult, dr, err); err != nil {
			t.Error(err)
		}
	}
}

func assertError(expectedErr string, actualErr error) error {
	if actualErr == nil && expectedErr == "" {
		return nil
	}
	if actualErr == nil {
		return fmt.Errorf("got nil; want error %q", expectedErr)
	}
	actual := actualErr.Error()
	if actual != expectedErr {
		return fmt.Errorf("got error %q; want %q", actual, expectedErr)
	}
	return nil
}

func assertResult(expected *sous.DetectResult, actual *sous.DetectResult, err error) error {
	if actual == nil && expected == nil {
		return nil
	}
	if (actual == nil && expected != nil) || (actual != nil && expected == nil) {
		return fmt.Errorf("got %#v; want %#v", actual, expected)
	}
	if actual.Compatible != expected.Compatible {
		return fmt.Errorf("Compatible = %v; want %v", actual.Compatible, expected.Compatible)
	}
	if (actual.Data == nil && expected.Data != nil) || (actual.Data != nil && expected.Data == nil) {
		return fmt.Errorf("Data = %#v; want %#v", actual.Data, expected.Data)
	}
	ad := actual.Data.(detectData)
	ed := expected.Data.(detectData)
	if ad != ed {
		return fmt.Errorf("Data = %#v; want %#v", ad, ed)
	}
	return nil
}
