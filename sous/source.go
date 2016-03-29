package sous

type (
	// SourceContext contains contextual information about the source code being
	// built.
	SourceContext struct {
		Branch, Revision, OffsetDir  string
		Files                        []string
		NearestTag, NearestSemverTag Tag
		DirtyWorkingTree             bool
	}
)
