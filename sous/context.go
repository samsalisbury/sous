package sous

import "io"

type (
	SourceContext interface {
		WorkDir() string
		Repo() RepoContext
	}

	Path    string
	WorkDir Path
	Output  io.Writer

	RepoContext struct {
		RepoRootDir,
		RepoOffsetDir,
		CommitID,
		Branch string
	}

	ChangesSinceLastBuild struct {
		SousUpdated bool
		NewCommit,
		NewTag string
		NewFiles, ChangedFiles []string
	}
)
