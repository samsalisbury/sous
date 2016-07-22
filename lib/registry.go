package sous

type (
	Registry interface {
		// GetArtifact gets the build artifact address for a source ID.
		// It does not guarantee that that artifact exists.
		GetArtifact(SourceID) (*BuildArtifact, error)
		// GetSourceID gets the source ID associated with the
		// artifact, regardless of the existence of the artifact.
		GetSourceID(*BuildArtifact) (SourceID, error)
		// GetMetadata returns metadata for a source ID.
		//GetMetadata(SourceID) (map[string]string, error)
	}
)
