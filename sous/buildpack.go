package sous

import "time"

type (
	// Buildpack is a set of instructions used to build a particular
	// kind of project.
	Buildpack interface {
		Build(*BuildContext) (*BuildResult, error)
	}
	// BuildResult represents the result of a build made with a Buildpack.
	BuildResult struct {
		ImageName string
		Elapsed   time.Duration
	}
)
