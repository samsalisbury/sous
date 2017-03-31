package docker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
)

func testSBPDetect(dockerfile string) (*sous.DetectResult, error) {
	scp := &SplitBuildpack{}
	ctx := &sous.BuildContext{}

	testDir, err := ioutil.TempDir("testdata")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(testDir)

	sh, err := shell.DefaultInDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	c := &sous.BuildContext{
		Sh:     sh,
		Source: sous.SourceContext{},
	}
	if dockerfile != "" {
		dockerfilePath := filepath.Join(testDir, "Dockerfile")
		if err := ioutil.WriteFile(dockerfilePath, []byte(dockerfile), 0777); err != nil {
			t.Fatal(err)
		}
	}

	return (&SplitBuildpack{}).Detect(c)
}

func assertRejected(t *testing.T, drez *sous.DetectResult, err error) {
	if drez.Compatible {
		t.Errorf("SplitBuildpack incorrectly reported compatible project: %#v", drez)
	}
	if err != nil {
		t.Errorf("SplitBuildpack returned unexpected error: %#v", err)
	}
}

func TestSplitBuildpackDetect(t *testing.T) {
	assertRejected(testSBPDetect(""))
	assertRejected(testSBPDetect("ENV SOUS_MANIFEST=/sous-manifest.json"))
}
