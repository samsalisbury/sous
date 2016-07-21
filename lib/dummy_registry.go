package sous

// DummyNameCache implements the Builder interface by returning a
// computed image name for a given source version
type DummyRegistry struct {
}

// NewDummyNameCache builds a new DummyNameCache
func NewDummyRegistry() *DummyRegistry {
	return &DummyRegistry{}
}

func (dc *DummyRegistry) GetArtifact(sv SourceID) (*BuildArtifact, error) {
	return &BuildArtifact{Name: sv.String(), Type: "dummy"}, nil
}

// GetSourceVersion implements part of ImageMapper
func (dc *DummyRegistry) GetSourceVersion(*BuildArtifact) (SourceID, error) {
	return SourceID{}, nil
}
