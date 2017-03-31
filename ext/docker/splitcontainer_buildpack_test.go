package docker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/shell"
)

func testSBPDetect(t *testing.T, dockerfile string,
	metadataMap map[string]docker_registry.Metadata) (*sous.DetectResult, error) {
	testDir, err := ioutil.TempDir("testdata", "splitcontainer")
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

	rc := docker_registry.NewDummyClient()
	for k, v := range metadataMap {
		rc.AddMetadata(k, v)
	}
	sbp := NewSplitBuildpack(rc)

	return (sbp).Detect(c)
}

func assertAccepted(t *testing.T, drez *sous.DetectResult, err error) {
	if err != nil {
		t.Errorf("SplitBuildpack reported unexpected error %#v", err)
	}
	if drez == nil {
		t.Errorf("SplitBuildpack returned a nil DetectResult")
	} else if !drez.Compatible {
		t.Errorf("SplitBuildpack incorrectly reported incompatible project: %#v", drez)
	}
}

func assertRejected(t *testing.T, drez *sous.DetectResult, err error) {
	if err != nil {
		return // an error implies rejection
	}
	if drez == nil {
		t.Errorf("SplitBuildpack returned a nil DetectResult")
	} else if drez.Compatible {
		t.Errorf("SplitBuildpack incorrectly reported compatible project: %#v", drez)
	}
}

func TestSplitBuildpackDetect(t *testing.T) {
	dr, err := testSBPDetect(t, "", nil)
	assertRejected(t, dr, err)

	dr, err = testSBPDetect(t, "ENV SOUS_BUILD_MANIFEST=/sous-manifest.json", nil)
	assertAccepted(t, dr, err)

	dr, err = testSBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3", nil)
	assertRejected(t, dr, err)

	dr, err = testSBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3",
		map[string]docker_registry.Metadata{".*blub-builder.*": {}})
	assertRejected(t, dr, err)

	dr, err = testSBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3",
		map[string]docker_registry.Metadata{
			".*blub-builder.*": {
				Env: map[string]string{"SOUS_BUILD_MANIFEST": "/sous-manifest.json"}},
		})
	assertAccepted(t, dr, err)
}
