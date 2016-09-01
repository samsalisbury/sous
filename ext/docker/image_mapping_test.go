package docker

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/nyarly/testify/require"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
)

func inMemoryDB(name string) *sql.DB {
	db, err := GetDatabase(&DBConfig{"sqlite3", InMemoryConnection(name)})
	if err != nil {
		panic(err)
	}
	return db
}

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryDB("roundtrip"))

	sv := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", "1.2.3")

	host := "docker.repo.io"
	base := "ot/wackadoo"
	in := base + ":version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	err := nc.insert(sv, in, digest, []sous.Quality{})
	assert.NoError(err)

	cn, err := nc.GetCanonicalName(in)
	if assert.NoError(err) {
		assert.Equal(in, cn)
	}
	nin, _, err := nc.getImageName(sv)
	if assert.NoError(err) {
		assert.Equal(in, nin)
	}

	newSV := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", "1.2.42")

	cn = base + "@" + digest
	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        Labels(newSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})
	sv, err = nc.GetSourceID(NewBuildArtifact(in, nil))
	if assert.Nil(err) {
		assert.Equal(newSV, sv)
	}

	ncn, err := nc.GetCanonicalName(host + "/" + in)
	if assert.Nil(err) {
		assert.Equal(host+"/"+cn, ncn)
	}
}

// I'm still exploring what the problem is here...
func TestHarvestAlso(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryDB("harvest_also"))

	host := "docker.repo.io"
	base := "ot/wackadoo"
	repo := "github.com/opentable/test-app"

	stuffBA := func(n, v string) sous.SourceID {
		ba := &sous.BuildArtifact{
			Name: n,
			Type: "docker",
		}

		sv := sous.MustNewSourceID(repo, "", v)

		in := base + ":version-" + v
		digBs := sha256.Sum256([]byte(in))
		digest := hex.EncodeToString(digBs[:])
		cn := base + "@sha256:" + digest

		dc.FeedMetadata(docker_registry.Metadata{
			Registry:      host,
			Labels:        Labels(sv),
			Etag:          digest,
			CanonicalName: cn,
			AllNames:      []string{cn, in},
		})
		sid, err := nc.GetSourceID(ba)
		assert.NoError(err)
		assert.NotNil(sid)
		return sid
	}
	sid1 := stuffBA("tom", "0.2.1")
	sid2 := stuffBA("dick", "0.2.2")
	sid3 := stuffBA("harry", "0.2.3")

	_, err := nc.GetArtifact(sid1) //which should not miss
	assert.NoError(err)
	_, err = nc.GetArtifact(sid2) //which should not miss
	assert.NoError(err)
	_, err = nc.GetArtifact(sid3) //which should not miss
	assert.NoError(err)
}

// This can happen e.g. if the same source gets built twice
func TestSecondCanonicalName(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryDB("secondCN"))

	host := "docker.repo.io"
	base := "ot/wackadoo"
	repo := "github.com/opentable/test-app"

	stuffBA := func(digest string) sous.SourceID {
		n := "test-service"
		v := `0.1.2-ci1234`
		ba := &sous.BuildArtifact{
			Name: n,
			Type: "docker",
		}

		sv := sous.MustNewSourceID(repo, "", v)

		in := base + ":version-" + v
		cn := base + "@sha256:" + digest

		dc.FeedMetadata(docker_registry.Metadata{
			Registry:      host,
			Labels:        Labels(sv),
			Etag:          digest,
			CanonicalName: cn,
			AllNames:      []string{cn, in},
		})
		sid, err := nc.GetSourceID(ba)
		if !assert.NoError(err) {
			fmt.Println(err)
			nc.dump(os.Stderr)
		}
		assert.NotNil(sid)
		return sid
	}
	sid1 := stuffBA(`012345678901234567890123456789AB012345678901234567890123456789AB`)
	sid2 := stuffBA(`ABCDEFABCDEFABCDEABCDEABCDEABCDEABCDEABCDEABCDEABCDEF12341234566`)

	_, err := nc.GetArtifact(sid1) //which should not miss
	assert.NoError(err)

	_, err = nc.GetArtifact(sid2) //which should not miss
	assert.NoError(err)
}

func TestHarvesting(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryDB("harvesting"))

	v := "1.2.3"
	sv := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", v)

	v2 := "2.3.4"
	sisterSV := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", v2)

	host := "docker.repo.io"
	base := "ot/wackadoo"
	tag := "version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	cn := base + "@" + digest
	in := base + ":" + tag

	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        Labels(sv),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	// a la a SetCollector getting the SV
	_, err := nc.GetSourceID(NewBuildArtifact(in, nil))
	if err != nil {
		fmt.Printf("%+v", err)
	}
	assert.Nil(err)

	tag = "version-2.3.4"
	dc.FeedTags([]string{tag})
	digest = "sha256:abcdefabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeffffffff"
	cn = base + "@" + digest
	in = base + ":" + tag
	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        Labels(sisterSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	nin, err := nc.GetArtifact(sisterSV)
	if assert.NoError(err) {
		assert.Equal(host+"/"+cn, nin.Name)
	}
}

func TestRecordAdvisories(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryDB("advisories"))
	v := "1.2.3"
	sv := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", v)
	base := "ot/wackadoo"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	cn := base + "@" + digest

	qs := []sous.Quality{{"ephemeral_tag", "advisory"}}

	err := nc.insert(sv, cn, digest, qs)
	assert.NoError(err)

	arty, err := nc.GetArtifact(sv)
	assert.NoError(err)
	require.NotNil(arty)
	require.Len(arty.Qualities, 1)
	assert.Equal(arty.Qualities[0].Name, `ephemeral_tag`)
}

func TestDump(t *testing.T) {
	assert := assert.New(t)

	io := &bytes.Buffer{}

	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryDB("dump"))

	nc.dump(io)
	assert.Regexp(`name_id`, io.String())
}

func TestMissingName(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()
	nc := NewNameCache(dc, inMemoryDB("missing"))

	v := "4.5.6"
	sv := sous.MustNewSourceID("https://github.com/opentable/brand-new-idea", "nested/there", v)

	name, _, err := nc.getImageName(sv)
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
