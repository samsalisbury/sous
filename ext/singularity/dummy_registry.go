package singularity

import "github.com/opentable/sous/lib"

// DummyNameCache implements the Builder interface by returning a
// computed image name for a given source version
type DummyRegistry struct {
}

// NewDummyNameCache builds a new DummyNameCache
func NewDummyRegistry() *DummyRegistry {
	return &DummyRegistry{}
}

// TODO: Factor out name cache concept from core sous lib & get rid of this func.
func (dc *DummyRegistry) GetArtifact(sv sous.SourceVersion) (*sous.BuildArtifact, error) {
	return &sous.BuildArtifact{Name: sv.String(), Type: "dummy"}, nil
}

// GetSourceVersion implements part of ImageMapper
func (dc *DummyRegistry) GetSourceVersion(*sous.BuildArtifact) (sous.SourceVersion, error) {
	return sous.SourceVersion{}, nil
}
