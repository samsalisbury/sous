package actions

import (
	"fmt"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// Build handles building deployable artifacts.
type Build struct {
	GetArtifact  *GetArtifact
	BuildManager *sous.BuildManager
	RFF          *sous.ResolveFilter
	CLIArgs      []string

	result *sous.BuildResult
}

// Result returns the result of this build.
func (sb *Build) Result() *sous.BuildResult {
	return sb.result
}

// Do performs the build.
func (sb *Build) Do() error {
	if len(sb.CLIArgs) != 0 {
		if err := sb.BuildManager.OffsetFromWorkdir(sb.CLIArgs[0]); err != nil {
			return cmdr.EnsureErrorResult(err)
		}
	}

	registered, err := sb.GetArtifact.ArtifactExists()
	if err != nil {
		return fmt.Errorf("unable to verify artifact existence: %s", err)
	}
	if registered {
		return fmt.Errorf("artifact already registered")
	}

	sb.result, err = sb.BuildManager.Build()
	return err
}
