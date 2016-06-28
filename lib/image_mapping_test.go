package sous

import (
	"log"
	"testing"

	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, "sqlite3", InMemoryConnection("roundtrip"))

	v := semv.MustParse("1.2.3")
	sv := SourceVersion{
		Version:    v,
		RepoURL:    RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: RepoOffset("nested/there"),
	}
	host := "docker.repo.io"
	base := "ot/wackadoo"
	in := base + ":version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
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

	cn = base + "@" + digest
	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        newSV.DockerLabels(),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})
	sv, err = nc.GetSourceVersion(in)
	if assert.Nil(err) {
		assert.Equal(newSV, sv)
	}

	ncn, err := nc.GetCanonicalName(host + "/" + in)
	if assert.Nil(err) {
		assert.Equal(host+"/"+cn, ncn)
	}
}

func TestHarvesting(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, "sqlite3", InMemoryConnection("roundtrip"))

	v := semv.MustParse("1.2.3")
	sv := SourceVersion{
		Version:    v,
		RepoURL:    RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: RepoOffset("nested/there"),
	}

	v2 := semv.MustParse("2.3.4")
	sisterSV := SourceVersion{
		Version:    v2,
		RepoURL:    RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: RepoOffset("nested/there"),
	}

	host := "docker.repo.io"
	base := "ot/wackadoo"
	tag := "version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	cn := base + "@" + digest
	in := base + ":" + tag

	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        sv.DockerLabels(),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	// a la a SetCollector getting the SV
	_, err := nc.GetSourceVersion(in)
	assert.Nil(err)

	tag = "version-2.3.4"
	dc.FeedTags([]string{tag})
	cn = base + "@" + digest
	in = base + ":" + tag
	digest = "sha256:abcdefabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdefffffffff"
	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        sisterSV.DockerLabels(),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	nin, err := nc.GetImageName(sisterSV)
	if assert.NoError(err) {
		assert.Equal(host+"/"+cn, nin)
	}
}

func TestMissingName(t *testing.T) {
	assert := assert.New(t)
	log.SetFlags(log.Flags() | log.Lshortfile)
	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, "sqlite3", InMemory)

	v := semv.MustParse("4.5.6")
	sv := SourceVersion{
		Version:    v,
		RepoURL:    RepoURL("https://github.com/opentable/brand-new-idea"),
		RepoOffset: RepoOffset("nested/there"),
	}

	name, err := nc.GetImageName(sv)
	assert.Equal("", name)
	assert.Error(err)
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
