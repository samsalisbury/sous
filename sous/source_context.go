package sous

type (
	// SourceContext contains contextual information about the source code being
	// built.
	SourceContext struct {
		RootDir, OffsetDir, Branch, Revision string
		Files, ModifiedFiles, NewFiles       []string
		Tags                                 []Tag
		NearestTagName                       string
		DirtyWorkingTree                     bool
	}
	// Tag represents a revision control commit tag.
	Tag struct {
		Name, Revision string
	}
)
