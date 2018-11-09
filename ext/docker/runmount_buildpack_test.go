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

func TestRunmountBuildpackDetect(t *testing.T) {
	dr, err := testRMBPDetect(t, "", nil)
	assertRejected(t, dr, err)

	dr, err = testRMBPDetect(t, `
	ENV SOUS_RUN_IMAGE_SPEC=/sous-manifest.json
	ENV BUILD_OUT=/build_output
	`, nil)
	assertAccepted(t, dr, err)
	assertArgs(t, dr, false, false)

	dr, err = testRMBPDetect(t, `
	ENV SOUS_RUN_IMAGE_SPEC=/sous-manifest.json
	ENV BUILD_OUT=/build_output
	ARG APP_VERSION
	ARG APP_REVISION
	`, nil)
	assertAccepted(t, dr, err)
	assertArgs(t, dr, true, true)

	dr, err = testRMBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3", nil)
	assertRejected(t, dr, err)

	dr, err = testRMBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3",
		map[string]docker_registry.Metadata{".*blub-builder.*": {}})
	assertRejected(t, dr, err)

	dr, err = testRMBPDetect(t, "FROM docker.opentable.com/blub-builder:1.2.3",
		map[string]docker_registry.Metadata{
			".*blub-builder.*": {
				Env: map[string]string{
					"SOUS_RUN_IMAGE_SPEC": "/sous-manifest.json",
					"BUILD_OUT":           "/build_output",
				}},
		})
	assertAccepted(t, dr, err)
	assertArgs(t, dr, false, false)

	dr, err = testRMBPDetect(t, `
	FROM docker.opentable.com/blub-builder:1.2.3
	ENV BUILD_OUT=/build_output
	`,
		map[string]docker_registry.Metadata{
			".*blub-builder.*": {
				Env: map[string]string{
					"SOUS_RUN_IMAGE_SPEC": "/sous-manifest.json",
				}},
		})
	assertAccepted(t, dr, err)
	assertArgs(t, dr, false, false)

	dr, err = testRMBPDetect(t, `
	FROM docker.opentable.com/blub-builder:1.2.3
	ARG APP_VERSION
	ARG APP_REVISION
	`,
		map[string]docker_registry.Metadata{
			".*blub-builder.*": {
				Env: map[string]string{
					"SOUS_RUN_IMAGE_SPEC": "/sous-manifest.json",
					"BUILD_OUT":           "/build_output",
				}},
		})
	assertAccepted(t, dr, err)
	assertArgs(t, dr, true, true)

	// n.b. Docker does not record ARG lines in containers, so there's no way for
	// the build container to expose APP_VERSION or APP_REVISION
	// Perhaps we should consider ENVs for those?
}

func testRMBPDetect(
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
	sbp := NewRunmountBuildpack(rc, logging.SilentLogSet())
	dr, err := sbp.Detect(c)

	return dr, err
}
