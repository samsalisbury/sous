package singularity

import (
	"crypto/sha1"

	sous "github.com/opentable/sous/lib"
)

type testRegistry struct {
	Images map[string]*testImage
}

// AddImage adds an image with name provided and with labels corresponding to
// repo, offset and tag.
func (tr *testRegistry) AddImage(name, repo, offset, tag string) {
	if offset != "" {
		offset = "," + offset
	}
	revision := string(sha1.New().Sum([]byte(name)))
	imageLabels := map[string]string{
		"com.opentable.sous.repo_url":    repo,
		"com.opentable.sous.version":     tag,
		"com.opentable.sous.revision":    revision,
		"com.opentable.sous.repo_offset": offset,
	}
	tr.Images[name] = &testImage{
		labels: imageLabels,
	}
}

func (tr *testRegistry) GetArtifact(sid sous.SourceID) (*sous.BuildArtifact, error) {
	panic("implements sous.Registry")
}

func (tr *testRegistry) GetSourceID(ba *sous.BuildArtifact) (sous.SourceID, error) {
	panic("implements sous.Registry")
}

func (tr *testRegistry) ImageLabels(imageName string) (map[string]string, error) {
	return tr.Images[imageName].labels, nil
}

func (tr *testRegistry) ListSourceIDs() ([]sous.SourceID, error) {
	panic("implements sous.Registry")
}

func (tr *testRegistry) Warmup(string) error {
	panic("implements sous.Registry")
}

type testImage struct {
	labels map[string]string
}

func newTestRegistry() *testRegistry {
	return &testRegistry{
		Images: map[string]*testImage{},
	}
}
