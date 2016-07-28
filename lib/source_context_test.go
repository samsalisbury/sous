package sous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert := assert.New(t)

	sc := SourceContext{
		OffsetDir:      "sub",
		RemoteURL:      "github.com/opentable/test",
		NearestTagName: "1.2.3",
	}
	id := sc.Version()
	assert.Equal("github.com/opentable/test", string(id.RepoURL))
	assert.Equal("sub", string(id.RepoOffset))
	assert.Equal("1.2.3", id.Version.String())
}
