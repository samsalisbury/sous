package sous

type (
	Registry interface {
		// GetArtifact gets the build artifact address for a named version.
		// It does not guarantee that that artifact exists.
		GetArtifact(SourceVersion) (*BuildArtifact, error)
		// GetSourceVersion gets the source version associated with the
		// artifact, regardless of the existence of the artifact.
		GetSourceVersion(*BuildArtifact) (SourceVersion, error)
		// GetMetadata returns metadata for source version.
		//GetMetadata(SourceVersion) (map[string]string, error)
	}
)
