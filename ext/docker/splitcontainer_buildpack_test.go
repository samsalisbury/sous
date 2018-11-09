package docker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/nyarly/spies"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/pkg/errors"
)

func TestSplitBuildpackDetect(t *testing.T) {
	dr, err := testSBPDetect(t, "", nil)
	assertRejected(t, dr, err)

	dr, err = testSBPDetect(t, "ENV SOUS_RUN_IMAGE_SPEC=/sous-manifest.json", nil)
	assertAccepted(t, dr, err)
	assertArgs(t, dr, false, false)

	dr, err = testSBPDetect(t, `
	ENV SOUS_RUN_IMAGE_SPEC=/sous-manifest.json
	ARG APP_VERSION
	ARG APP_REVISION
	`, nil)
	assertAccepted(t, dr, err)
	assertArgs(t, dr, true, true)

	dr, err = testSBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3", nil)
	assertRejected(t, dr, err)

	dr, err = testSBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3",
		map[string]docker_registry.Metadata{".*blub-builder.*": {}})
	assertRejected(t, dr, err)

	dr, err = testSBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3",
		map[string]docker_registry.Metadata{
			".*blub-builder.*": {
				Env: map[string]string{"SOUS_RUN_IMAGE_SPEC": "/sous-manifest.json"}},
		})
	assertAccepted(t, dr, err)
	assertArgs(t, dr, false, false)

	dr, err = testSBPDetect(t, `
	FROM docker.opentable.com/blub-builder:1.2.3
	ARG APP_VERSION
	ARG APP_REVISION
	`,
		map[string]docker_registry.Metadata{
			".*blub-builder.*": {
				Env: map[string]string{"SOUS_RUN_IMAGE_SPEC": "/sous-manifest.json"}},
		})
	assertAccepted(t, dr, err)
	assertArgs(t, dr, true, true)

	// n.b. Docker does not record ARG lines in containers, so there's no way for
	// the build container to expose APP_VERSION or APP_REVISION
	// Perhaps we should consider ENVs for those?
}

func testSBPDetect(
	t *testing.T,
	dockerfile string,
	metadataMap map[string]docker_registry.Metadata,
) (*sous.DetectResult, error) {
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
	rc.MatchMethod("GetImageMetadata", spies.AnyArgs, docker_registry.Metadata{}, errors.Errorf("no such MD"))
	sbp := NewSplitBuildpack(rc, logging.SilentLogSet())
	dr, err := sbp.Detect(c)

	return dr, err
}

func assertAccepted(t *testing.T, drez *sous.DetectResult, err error) {
	t.Helper()
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
	t.Helper()
	if err != nil {
		return // an error implies rejection
	}
	if drez == nil {
		t.Errorf("SplitBuildpack returned a nil DetectResult")
	} else if drez.Compatible {
		t.Errorf("SplitBuildpack incorrectly reported compatible project: %#v", drez)
	}
}

func assertArgs(t *testing.T, drez *sous.DetectResult, version, revision bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Error(r)
		}
	}()
	if drez.Data.(detectData).HasAppRevisionArg != revision {
		t.Errorf("Expected detected revision arg: %t, was: %t", revision, drez.Data.(detectData).HasAppRevisionArg)
	}
	if drez.Data.(detectData).HasAppVersionArg != version {
		t.Errorf("Expected detected version arg: %t, was: %t", version, drez.Data.(detectData).HasAppVersionArg)
	}
}
