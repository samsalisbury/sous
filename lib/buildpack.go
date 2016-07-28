package sous

import "time"

type (
	// A Selector selects the buildpack for a given build context
	Selector interface {
		SelectBuildpack(*BuildContext) (Buildpack, error)
	}

	// Labeller defines a container-based build system.
	Labeller interface {
		// Build performs a build and returns the result.
		//Build(*BuildContext, Buildpack, *DetectResult) (*BuildResult, error)
		ApplyMetadata(*BuildResult) error
	}

	// Registrar defines the interface to register build results to be deployed
	// later
	Registrar interface {
		// Register takes a BuildResult and makes it available for the deployment
		// target system to find during deployment
		Register(*BuildResult) error
	}

	// BuildArtifact describes the actual built binary Sous will deploy
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
		ImageID                   string
		VersionName, RevisionName string
		Advisories                []string
		Elapsed                   time.Duration
	}

	EchoSelector struct {
		Factory func(*BuildContext) (Buildpack, error)
	}
)

func (s *EchoSelector) SelectBuildpack(c *BuildContext) (Buildpack, error) {
	return s.Factory(c)
}
