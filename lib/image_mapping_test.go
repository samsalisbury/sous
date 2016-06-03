package sous

import (
	"strings"
	"testing"

	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, "sqlite3", ":memory:")

	v := semv.MustParse("1.2.3")
	sv := SourceVersion{
		Version:    v,
		RepoURL:    RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: RepoOffset("nested/there"),
	}
	base := "docker.repo.io/wackadoo"
	in := base + ":version-1.2.3"
	digest := "sha256:deadbeef1234567890"
	err := nc.Insert(sv, in, digest)
	assert.NoError(err)

	cn, err := nc.GetCanonicalName(in)
	if assert.NoError(err) {
		assert.Equal(in, cn)
	}
	nin, err := nc.GetImageName(sv)
	if assert.NoError(err) {
		assert.Equal(in, nin)
	}

	newV := semv.MustParse("1.2.42")
	newSV := SourceVersion{
		Version:    newV,
		RepoURL:    RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: RepoOffset("nested/there"),
	}

	cn = strings.Join([]string{base, "@", digest}, "")
	dc.FeedMetadata(docker_registry.Metadata{
		Labels:        newSV.DockerLabels(),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})
	sv, err = nc.GetSourceVersion(in)
	if assert.Nil(err) {
		assert.Equal(newSV, sv)
	}

	ncn, err := nc.GetCanonicalName(in)
	if assert.Nil(err) {
		assert.Equal(cn, ncn)
	}

}

func TestUnion(t *testing.T) {
	assert := assert.New(t)

	left := []string{"a", "b", "c"}
	right := []string{"b", "c", "d"}

	all := union(left, right)
	assert.Equal(len(all), 4)
	assert.Contains(all, "a")
	assert.Contains(all, "b")
	assert.Contains(all, "c")
	assert.Contains(all, "d")
}
