package sous

import "os/user"

type (
	// BuildContext contains all the data required to perform a build.
	BuildContext struct {
		Source  SourceContext
		Scratch ScratchContext
		Machine Machine
		User    user.User
		Changes Changes
	}
	// SourceContext contains contextual information about the source code being
	// built.
	SourceContext struct {
		Branch, Revision, OffsetDir  string
		Files                        []string
		NearestTag, NearestSemverTag Tag
		DirtyWorkingTree             bool
	}
	// ScratchContext represents an isolated copy of a project's source code
	// somewhere on the host machine running Sous.
	ScratchContext struct {
		RootDir, OffsetDir string
	}
	// Tag represents a revision control commit tag.
	Tag struct {
		Name, Revision string
	}
	// Machine represents a specific computer.
	Machine struct {
		Host, FullHost string
	}
	// Changes represents a set of changes that have happened since this project
	// was last built on the current machine by the current user.
	Changes struct {
		SousUpdated, NewCommit, NewTag, NewFiles, ChangedFiles []string
	}
)
