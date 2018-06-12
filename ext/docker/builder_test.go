package docker

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataDockerfile(t *testing.T) {
	assert := assert.New(t)

	b := Builder{}

	bp := sous.BuildProduct{
		ID:         "identifier",
		Advisories: []string{`something is horribly wrong`},
		Source:     sous.MakeSourceID("github.com/opentable/test", "sub", "2.3.7+abcd"),
	}
	mddf, err := ioutil.ReadAll(b.metadataDockerfile(&bp))

	assert.NoError(err)
	assert.Equal(
		// 991
		`FROM identifier
LABEL \
  com.opentable.sous.repo_offset="sub" \
  com.opentable.sous.repo_url="github.com/opentable/test" \
  com.opentable.sous.revision="" \
  com.opentable.sous.version="2.3.7" \
  com.opentable.sous.advisories="something is horribly wrong"`, string(mddf))
}

func TestTagStrings(t *testing.T) {
	assert := assert.New(t)

	sid, err := sous.NewSourceID("github.com/opentable/sous", "docker", "1.2.3+deadbeef")
	if err != nil {
		t.Fatal(err)
	}

	theTime := time.Unix(212742000, 0)

	ls := logging.SilentLogSet()

	assert.Equal("sous/docker:1.2.3", versionName(sid, "", stripRE))
	assert.Equal("sous/docker-builder:1.2.3", versionName(sid, "builder", stripRE))
	assert.Equal("sous/docker:zdeadbeef-1976-09-28T07.00.00", revisionName(sid, "deadbeef", "", theTime, stripRE))
	assert.Equal("sous/docker-builder:zdeadbeef-1976-09-28T07.00.00", revisionName(sid, "deadbeef", "builder", theTime, stripRE))
	assert.Equal("docker.example.com/sous/docker:1.2.3", versionTag("docker.example.com", sid, "", stripRE, ls))
	assert.Equal("docker.example.com/sous/docker-builder:zdeadbeef-1976-09-28T07.00.00", revisionTag("docker.example.com", sid, "deadbeef", "builder", theTime, stripRE, ls))
}

func TestBuilderApplyMetadata(t *testing.T) {
	srcSh, srcCtl := shell.NewTestShell()

	scratchSh, _ := shell.NewTestShell()

	nc, _ := sous.NewInserterSpy()

	b, err := NewBuilder(nc, "docker.example.com", srcSh, scratchSh, logging.SilentLogSet())
	require.NoError(t, err)

	br := &sous.BuildResult{
		Products: []*sous.BuildProduct{
			{},
			{Kind: "builder"},
		},
	}

	err = b.ApplyMetadata(br)
	assert.NoError(t, err)

	err = b.Register(br)
	assert.NoError(t, err)
	assert.Len(t, nc.CallsTo("Insert"), 2)

	assert.Len(t, srcCtl.CmdsLike("docker", "push"), 4)
}
