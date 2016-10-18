package sous

type (
	// Registry describes a system for mapping SourceIDs to BuildArtifacts and vice versa
	Registry interface {
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

		//ImageLabels finds the sous (docker) labels for a given image name
		ImageLabels(imageName string) (labels map[string]string, err error)
	}

	// An Inserter puts data into a registry.
	Inserter interface {
		// Insert pairs a SourceID with an imagename, and tags the pairing with Qualities
		// The etag can be (usually will be) the empty string
		Insert(sid SourceID, in, etag string, qs []Quality) error
	}
)
