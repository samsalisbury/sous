package sous

type (
	// Registry describes a system for mapping SourceIDs to BuildArtifacts and vice versa
	Registry interface {
		// GetAdvisories gets all the advisories on the image associated with sourceID.
		GetAdvisories(SourceID) (Advisories, error)
		// GetArtifact gets the build artifact address for a source ID.
		// It does not guarantee that that artifact exists.
		GetArtifact(SourceID) (*BuildArtifact, error)
		// GetSourceID gets the source ID associated with the
		// artifact, regardless of the existence of the artifact.
		GetSourceID(*BuildArtifact) (SourceID, error)
		// GetMetadata returns metadata for a source ID.
		//GetMetadata(SourceID) (map[string]string, error)

		// ListSourceIDs returns a list of known SourceIDs
		ListSourceIDs() ([]SourceID, error)

		// Warmup requests that the registry check specific artifact names for existence
		// the details of this behavior will vary by implementation. For Docker, for instance,
		// the corresponding repo is enumerated
		Warmup(string) error
	}
)
