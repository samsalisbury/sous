package sous

import (
	"strings"
	"testing"

	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

type mdChan chan docker_registry.Metadata

type DummyRegistryClient struct {
	mds mdChan
}

func (drc *DummyRegistryClient) Cancel()                  {}
func (drc *DummyRegistryClient) BecomeFoolishlyTrusting() {}

func (drc *DummyRegistryClient) GetImageMetadata(in, et string) (docker_registry.Metadata, error) {
	return <-drc.mds, nil
}

func (drc *DummyRegistryClient) LabelsForImageName(in string) (map[string]string, error) {
	md := <-drc.mds
	return md.Labels, nil
}

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)

	mds := make(mdChan, 10)
	nc := NewNameCache(&DummyRegistryClient{mds})

	v := semv.MustParse("1.2.3")
	sv := SourceVersion{
		Version:    v,
		RepoURL:    RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: RepoOffset("nested/there"),
	}
	base := "docker.repo.io/wackadoo"
	in := base + ":version-1.2.3"
	digest := "sha256:deadbeef1234567890"
	nc.Insert(sv, in, digest)

	cn, err := nc.GetCanonicalName(in)
	if assert.Nil(err) {
		assert.Equal(in, cn)
	}
	nin, err := nc.GetImageName(sv)
	if assert.Nil(err) {
		assert.Equal(in, nin)
	}

	newV := semv.MustParse("1.2.42")
	newSV := SourceVersion{
		Version:    newV,
		RepoURL:    RepoURL("https://github.com/opentable/wackadoo"),
		RepoOffset: RepoOffset("nested/there"),
	}

	cn = strings.Join([]string{base, "@", digest}, "")
	mds <- docker_registry.Metadata{
		Labels:        newSV.DockerLabels(),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	}
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
