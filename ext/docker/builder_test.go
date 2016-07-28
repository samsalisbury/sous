package docker

import (
	"io/ioutil"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/stretchr/testify/assert"
)

func TestMetadataDockerfile(t *testing.T) {
	assert := assert.New(t)

	b := Builder{
		Context: &sous.SourceContext{
			OffsetDir:      "sub",
			RemoteURL:      "github.com/opentable/test",
			Revision:       "abcd",
			NearestTagName: "2.3.7",
		},
	}

	br := sous.BuildResult{
		ImageID:    "identifier",
		Advisories: []string{`something is horribly wrong`},
	}
	mddf, err := ioutil.ReadAll(b.metadataDockerfile(&br))

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
