package sous

type (
	// DummyNameCache implements the Builder interface by returning a
	// computed image name for a given source ID.
	DummyRegistry struct {
		ars  chan artifactReturn
		sids chan sourceIDReturn
		ls   chan sourceIDListReturn
	}

	artifactReturn struct {
		*BuildArtifact
		error
	}
	sourceIDReturn struct {
		SourceID
		error
	}
	sourceIDListReturn struct {
		ids []SourceID
		error
	}
)

// NewDummyRegistry builds a new DummyNameCache.
func NewDummyRegistry() *DummyRegistry {
	return &DummyRegistry{
		ars:  make(chan artifactReturn, 20),
		sids: make(chan sourceIDReturn, 20),
		ls:   make(chan sourceIDListReturn, 20),
	}
}

func (dc *DummyRegistry) FeedArtifact(ba *BuildArtifact, e error) {
	dc.ars <- artifactReturn{ba, e}
}

func (dc *DummyRegistry) GetArtifact(sid SourceID) (*BuildArtifact, error) {
	select {
	case ar := <-dc.ars:
		return ar.BuildArtifact, ar.error
	default:
		return &BuildArtifact{Name: sid.String(), Type: "dummy"}, nil
	}
}

func (dc *DummyRegistry) FeedSourceID(sid SourceID, e error) {
	dc.sids <- sourceIDReturn{sid, e}
}

// GetSourceID implements part of ImageMapper
func (dc *DummyRegistry) GetSourceID(*BuildArtifact) (SourceID, error) {
	select {
	case sr := <-dc.sids:
		return sr.SourceID, sr.error
	default:
		return SourceID{}, nil
	}
}

func (dc *DummyRegistry) FeedSourceIDList(sids []SourceID, e error) {
	dc.ls <- sourceIDListReturn{sids, e}
}

// ListSourceIDs implements Registry
func (dc *DummyRegistry) ListSourceIDs() ([]SourceID, error) {
	select {
	case lr := <-dc.ls:
		return lr.ids, lr.error
	default:
		return []SourceID{}, nil
	}
}

// Warmup implements Registry
func (dc *DummyRegistry) Warmup(string) error {
	return nil
}
