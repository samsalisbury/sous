package docker

import (
	"database/sql"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func inMemoryRoundtripDB() *sql.DB {
	db, err := GetDatabase(&DBConfig{"sqlite3", InMemoryConnection("roundtrip")})
	if err != nil {
		panic(err)
	}
	return db
}

func inMemoryDB() *sql.DB {
	db, err := GetDatabase(&DBConfig{"sqlite3", InMemory})
	if err != nil {
		panic(err)
	}
	return db
}

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryRoundtripDB())

	v := semv.MustParse("1.2.3")
	sv := sous.SourceID{
		Version:    v,
		RepoURL:    sous.RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: sous.RepoOffset("nested/there"),
	}
	host := "docker.repo.io"
	base := "ot/wackadoo"
	in := base + ":version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	err := nc.insert(sv, in, digest)
	assert.NoError(err)

	cn, err := nc.GetCanonicalName(in)
	if assert.NoError(err) {
		assert.Equal(in, cn)
	}
	nin, err := nc.getImageName(sv)
	if assert.NoError(err) {
		assert.Equal(in, nin)
	}

	newV := semv.MustParse("1.2.42")
	newSV := sous.SourceID{
		Version:    newV,
		RepoURL:    sous.RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: sous.RepoOffset("nested/there"),
	}

	cn = base + "@" + digest
	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        DockerLabels(newSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})
	sv, err = nc.GetSourceVersion(DockerBuildArtifact(in))
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
	nc := NewNameCache(dc, inMemoryRoundtripDB())

	v := semv.MustParse("1.2.3")
	sv := sous.SourceID{
		Version:    v,
		RepoURL:    sous.RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: sous.RepoOffset("nested/there"),
	}

	v2 := semv.MustParse("2.3.4")
	sisterSV := sous.SourceID{
		Version:    v2,
		RepoURL:    sous.RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: sous.RepoOffset("nested/there"),
	}

	host := "docker.repo.io"
	base := "ot/wackadoo"
	tag := "version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	cn := base + "@" + digest
	in := base + ":" + tag

	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        DockerLabels(sv),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	// a la a SetCollector getting the SV
	_, err := nc.GetSourceVersion(DockerBuildArtifact(in))
	assert.Nil(err)

	tag = "version-2.3.4"
	dc.FeedTags([]string{tag})
	cn = base + "@" + digest
	in = base + ":" + tag
	digest = "sha256:abcdefabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdefffffffff"
	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        DockerLabels(sisterSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	nin, err := nc.getImageName(sisterSV)
	if assert.NoError(err) {
		assert.Equal(host+"/"+cn, nin)
	}
}

func TestMissingName(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryDB())

	v := semv.MustParse("4.5.6")
	sv := sous.SourceID{
		Version:    v,
		RepoURL:    sous.RepoURL("https://github.com/opentable/brand-new-idea"),
		RepoOffset: sous.RepoOffset("nested/there"),
	}

	name, err := nc.getImageName(sv)
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
