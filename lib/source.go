package sous

// Source represents a body of source code.
type Source struct {
	// ID is the identity of this source.
	ID SourceID
	// Context contains info about this repo
	Context SourceContext
	// LocalRootDir is the absolute path on the local filesystem to the cloned
	// repository root.
	LocalRootDir string
	// LocalOffsetDir is the absolute path on the local filesystem to the offset
	// matching ID.Location.Dir.
	LocalOffsetDir string
}
