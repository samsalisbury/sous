package sous

type (
	// SourceLocation identifies a directory inside a specific source code repo.
	// Note that the directory has no meaning without the addition of a revision
	// ID. This type is used as a shorthand for deploy manifests, enabling the
	// logical grouping of deploys of different versions of a particular
	// service.
	SourceLocation struct {
		// RepoURL is the URL of a source code repository.
		RepoURL RepoURL
		// RepoOffset is a relative path to a directory within the repository
		// at RepoURL
		RepoOffset `yaml:",omitempty"`
	}
)
