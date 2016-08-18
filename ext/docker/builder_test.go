package docker

import (
	"io/ioutil"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestMetadataDockerfile(t *testing.T) {
	assert := assert.New(t)

	b := Builder{}

	br := sous.BuildResult{
		ImageID:    "identifier",
		Advisories: []string{`something is horribly wrong`},
	}
	bc := sous.BuildContext{
		Source: sous.SourceContext{
			OffsetDir:      "sub",
			RemoteURL:      "github.com/opentable/test",
			Revision:       "abcd",
			NearestTagName: "2.3.7",
		},
	}
	mddf, err := ioutil.ReadAll(b.metadataDockerfile(&br, &bc))

	assert.NoError(err)
	assert.Equal(
		`FROM identifier
LABEL \
  com.opentable.sous.repo_offset="sub" \
  com.opentable.sous.repo_url="github.com/opentable/test" \
  com.opentable.sous.revision="abcd" \
  com.opentable.sous.version="2.3.7" \
  com.opentable.sous.advisories="something is horribly wrong,"`, string(mddf))
}

func TestTagStrings(t *testing.T) {
	assert := assert.New(t)

	sid := sous.SourceID{
		Repo:    "github.com/opentable/sous",
		Offset:  "docker",
		Version: semv.MustParse("1.2.3+deadbeef"),
	}

	assert.Equal("/sous/docker:1.2.3", versionName(sid))
	assert.Equal("/sous/docker:deadbeef", revisionName(sid))

}
