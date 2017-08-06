package docker

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
	"github.com/opentable/sous/util/spies"
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
		`FROM identifier
LABEL \
  com.opentable.sous.repo_offset="sub" \
  com.opentable.sous.repo_url="github.com/opentable/test" \
  com.opentable.sous.revision="abcd" \
  com.opentable.sous.version="2.3.7" \
  com.opentable.sous.advisories="something is horribly wrong"`, string(mddf))
}

func TestTagStrings(t *testing.T) {
	assert := assert.New(t)

	sid, err := sous.NewSourceID("github.com/opentable/sous", "docker", "1.2.3+deadbeef")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal("/sous/docker:1.2.3", versionName(sid, ""))
	assert.Equal("/sous/docker-builder:1.2.3", versionName(sid, "builder"))
	assert.Equal("/sous/docker:zdeadbeef-1976-09-28T07.00.00", revisionName(sid, "", time.Unix(212742000, 0)))
	assert.Equal("/sous/docker-builder:zdeadbeef-1976-09-28T07.00.00", revisionName(sid, "builder", time.Unix(212742000, 0)))
}

func TestBuilderApplyMetadata(t *testing.T) {
	srcSh, srcCtl := shell.NewTestShell()

	scratchSh, scratchCtl := shell.NewTestShell()

	scratchCtl.Any("List", []os.FileInfo{}, nil)

	nc := sous.NewInserterSpy()
	nc.Match(spies.Always, nil)

	b, err := NewBuilder(nc, "docker.example.com", srcSh, scratchSh)
	require.NoError(t, err)

	br := &sous.BuildResult{
		Products: []*sous.BuildProduct{
			{},
			{Kind: "builder"},
		},
	}

	buildcmd, buildctl := srcCtl.CmdFor("docker", "build")
	buildctl.Any("SetStdin")
	spew.Dump(buildcmd)

	err = b.ApplyMetadata(br)
	assert.NoError(t, err)

	err = b.Register(br)
	assert.NoError(t, err)
	assert.Len(t, nc.CallsTo("Insert"), 2)

	assert.Len(t, srcCtl.CmdsLike("docker", "push"), 4)
}
