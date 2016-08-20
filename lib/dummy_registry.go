package sous

// DummyNameCache implements the Builder interface by returning a
// computed image name for a given source ID.
type DummyRegistry struct {
}

// NewDummyNameCache builds a new DummyNameCache.
func NewDummyRegistry() *DummyRegistry {
	return &DummyRegistry{}
}

func (dc *DummyRegistry) GetArtifact(sid SourceID) (*BuildArtifact, error) {
	return &BuildArtifact{Name: sid.String(), Type: "dummy"}, nil
}

// GetSourceID implements part of ImageMapper
func (dc *DummyRegistry) GetSourceID(*BuildArtifact) (SourceID, error) {
	return SourceID{}, nil
}

// ListSourceIDs implements Registry
func (dc *DummyRegistry) ListSourceIDs() ([]SourceID, error) {
	return []SourceID{}, nil
}

// Warmup implements Registry
func (dc *DummyRegistry) Warmup(string) error {
	return nil
}
