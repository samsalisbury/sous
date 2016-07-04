package sous

import "time"

type (
	// Builder defines a container-based build system.
	Builder interface {
		// Build performs a build and returns the result.
		Build(*BuildContext, Buildpack, *DetectResult) (*BuildResult, error)
		// GetArtifact gets the build artifact address for a named version.
		// It does not guarantee that that artifact exists.
		GetArtifact(SourceVersion) (*BuildArtifact, error)
		// GetSourceVersion gets the source version associated with the
		// artifact.
		GetSourceVersion(*BuildArtifact) (SourceVersion, error)
	}
	BuildArtifact struct {
		Name, Type string
	}
	// Buildpack is a set of instructions used to build a particular
	// kind of project.
	Buildpack interface {
		Detect(*BuildContext) (*DetectResult, error)
		Build(*BuildContext) (*BuildResult, error)
	}
	// DetectResult represents the result of a detection.
	DetectResult struct {
		Compatible  bool
		Description string
		Data        interface{}
	}
	// BuildResult represents the result of a build made with a Buildpack.
	BuildResult struct {
		ImageID   string
		ImageName string
		Elapsed   time.Duration
	}
)
