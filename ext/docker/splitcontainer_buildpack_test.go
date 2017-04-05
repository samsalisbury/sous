package docker

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

func assertArgs(t *testing.T, drez *sous.DetectResult, version, revision bool) {
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

func TestSplitBuildpackDetect(t *testing.T) {
	dr, err := testSBPDetect(t, "", nil)
	assertRejected(t, dr, err)

	dr, err = testSBPDetect(t, "ENV SOUS_BUILD_MANIFEST=/sous-manifest.json", nil)
	assertAccepted(t, dr, err)
	assertArgs(t, dr, false, false)

	dr, err = testSBPDetect(t, `
	ENV SOUS_BUILD_MANIFEST=/sous-manifest.json
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
				Env: map[string]string{"SOUS_BUILD_MANIFEST": "/sous-manifest.json"}},
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
				Env: map[string]string{"SOUS_BUILD_MANIFEST": "/sous-manifest.json"}},
		})
	assertAccepted(t, dr, err)
	assertArgs(t, dr, true, true)

	// n.b. Docker does not record ARG lines in containers, so there's no way for
	// the build container to expose APP_VERSION or APP_REVISION
	// Perhaps we should consider ENVs for those?
}

func TestSplitBuildpackBuildTemplating(t *testing.T) {
	sb := &splitBuilder{
		Manifest: &SplitBuildManifest{
			Container: sbmContainer{From: "scratch"},
			Files: []sbmInstall{
				{Source: sbmFile{Dir: "src"}, Destination: sbmFile{Dir: "dest"}},
			},
			Exec: []string{"cat", "/etc/shadow"},
		},
		VersionConfig:  "APP_VERSION=1.2.3",
		RevisionConfig: "APP_REVISION=cabbagedeadbeef",
	}
	buf := &bytes.Buffer{}

	err := sb.templateDockerfileBytes(buf)
	if err != nil {
		t.Error(err)
	}
	dockerfile := buf.String()
	hasString := func(needle string) {
		if strings.Index(dockerfile, needle) == -1 {
			t.Errorf("No %q in dockerfile.", needle)
		}
	}
	hasString("FROM")
	hasString("APP_VERSION")
	hasString("COPY dest dest")
}

func TestSplitBuildpackBuildLoadManifest(t *testing.T) {
	sb := &splitBuilder{}

	mBuf := bytes.NewBufferString(`{
  "container": {
    "type": "Docker",
    "from": "scratch"
  },
  "files": [
    {
      "source": { "dir": "/built"},
      "dest":   { "dir": "/"}
    }
  ],
  "exec": ["/sous-demo"]
}`)

	sb.Manifest = &SplitBuildManifest{}
	dec := json.NewDecoder(mBuf)
	dec.Decode(sb.Manifest)

	if sb.Manifest.Container.From != "scratch" {
		t.Error("Manifest didn't load Container.From")
	}

	if len(sb.Manifest.Files) != 1 {
		t.Error("No files loaded")
	} else {
		if sb.Manifest.Files[0].Source.Dir != "/built" {
			t.Error("Manifest didn't load Container.Files[0].Source")
		}
		if sb.Manifest.Files[0].Destination.Dir != "/" {
			t.Error("Manifest didn't load Container.Files[0].Destination")
		}
	}
}
