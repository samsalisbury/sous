package sous

import (
	"log"
	"regexp"
	"testing"

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

	dockerID := "1234512345"
	tagStr := "docker.wearenice.com/awesomeproject/:1.2.3+987654321987654312"

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

	br, err := RunBuild("docker.wearenice.com", sourceCtx, sourceSh, scratchSh)
	assert.NotNil(br)
	assert.NoError(err)
	assert.Equal(len(sourceSh.History), 3)
	assert.Regexp("^"+regexp.QuoteMeta("docker build ."), sourceSh.History[0])

	assert.Regexp("^"+regexp.QuoteMeta("docker build -t "+tagStr)+".*", sourceSh.History[1])
	assert.Regexp("FROM "+dockerID, sourceSh.History[1].StdinString())
	assert.Regexp("com.opentable.sous.repo_url=github.com/opentable/awesomeproject", sourceSh.History[1].StdinString())

	assert.Regexp("^"+regexp.QuoteMeta("docker push "+tagStr), sourceSh.History[2])
}
