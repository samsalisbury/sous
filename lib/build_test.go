package sous

import (
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

func testSourceContext() *SourceContext {
	return &SourceContext{
		PossiblePrimaryRemoteURL: "github.com/opentable/awesomeproject",
		NearestTagName:           "1.2.3",
		Revision:                 "987654321987654312",
	}
}

func TestBuild(t *testing.T) {
	assert := assert.New(t)
	log.SetFlags(log.Flags() | log.Lshortfile)
	Log.Debug.SetOutput(os.Stderr)
	Log.Vomit.SetOutput(os.Stderr)

	repoName := "github.com/opentable/awesomeproject"
	revision := "987654321987654312"
	version := "1.2.3"

	sourceCtx := &SourceContext{
		PossiblePrimaryRemoteURL: repoName,
		NearestTagName:           version,
		Revision:                 revision,
	}

	dockerID := "1234512345"
	tagStr := "awesomeproject:"
	dockerHost := "docker.wearenice.com"
	versionName := dockerHost + "/" + tagStr + version
	revisionName := dockerHost + "/" + tagStr + revision

	sourceDir := "/home/jenny-dev/project"
	sourceFiles := map[string]string{
		"Dockerfile": "FROM base",
	}

	sourceSh, err := shell.NewTestShell(sourceDir, sourceFiles)
	if err != nil {
		log.Fatal(err)
	}

	tmpDir := "/tmp/1234deadbeef"
	tmpFiles := map[string]string{
		"__exists__": "",
	}
	scratchSh, err := shell.NewTestShell(tmpDir, tmpFiles)
	if err != nil {
		log.Fatal(err)
	}

	sourceSh.CmdsF = func(name string, args []interface{}) *shell.DummyResult {
		if name == "docker" && len(args) > 0 && args[0] == "build" {
			return &shell.DummyResult{
				SO: []byte("Successfully built " + dockerID),
			}
		}
		return nil
	}

	docker := docker_registry.NewDummyClient()
	nc := NewNameCache(docker, "sqlite3", InMemory)

	br, err := RunBuild(nc, "docker.wearenice.com", sourceCtx, sourceSh, scratchSh)
	assert.NotNil(br)
	assert.NoError(err)
	assert.Equal(len(sourceSh.History), 4)

	reTail := `\s*(#.*)?$` //dummy commands include a #comment to that effect

	assert.Regexp("^"+regexp.QuoteMeta("docker build .")+reTail, sourceSh.History[0])

	assert.Regexp("^"+regexp.QuoteMeta("docker build -t "+versionName+" -t "+revisionName+" - ")+reTail, sourceSh.History[1])
	assert.Regexp("FROM "+dockerID, sourceSh.History[1].StdinString())
	assert.Regexp("com.opentable.sous.repo_url=github.com/opentable/awesomeproject", sourceSh.History[1].StdinString())

	assert.Regexp("^"+regexp.QuoteMeta("docker push "+versionName)+reTail, sourceSh.History[2])
	assert.Regexp("^"+regexp.QuoteMeta("docker push "+revisionName)+reTail, sourceSh.History[3])
	docker.FeedMetadata(docker_registry.Metadata{
		Registry: dockerHost,
		Labels: map[string]string{
			DockerVersionLabel:  "1.2.3",
			DockerRevisionLabel: revision,
			DockerPathLabel:     "",
			DockerRepoLabel:     repoName,
		},
		Etag:          "digest",
		CanonicalName: versionName,
		AllNames:      []string{tagStr},
	})
	sv, err := nc.GetSourceVersion(versionName)
	if assert.NoError(err) {
		assert.Equal(repoName, string(sv.Repo()))
	}
}
