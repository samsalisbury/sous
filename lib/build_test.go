package sous

import (
	"log"
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

	sourceCtx := testSourceContext()

	repoName := "github.com/opentable/awesomeproject"
	dockerID := "1234512345"
	tagStr := "docker.wearenice.com/awesomeproject:1.2.3"

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

	nc := NewNameCache(docker_registry.NewClient())

	br, err := RunBuild(&nc, "docker.wearenice.com", sourceCtx, sourceSh, scratchSh)
	assert.NotNil(br)
	assert.NoError(err)
	assert.Equal(len(sourceSh.History), 3)
	assert.Regexp("^"+regexp.QuoteMeta("docker build ."), sourceSh.History[0])

	assert.Regexp("^"+regexp.QuoteMeta("docker build -t "+tagStr)+".*", sourceSh.History[1])
	assert.Regexp("FROM "+dockerID, sourceSh.History[1].StdinString())
	assert.Regexp("com.opentable.sous.repo_url=github.com/opentable/awesomeproject", sourceSh.History[1].StdinString())

	assert.Regexp("^"+regexp.QuoteMeta("docker push "+tagStr)+`\s*(#.*)?$`, sourceSh.History[2])
	log.Printf("tagStr = %+v\n", tagStr)
	sv, err := nc.GetSourceVersion(tagStr)
	if assert.NoError(err) {
		assert.Equal(repoName, string(sv.Repo()))
	}
}
