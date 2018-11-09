package actions

import (
	"fmt"
	"strings"

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

	if err := assertArtifactNotRegistered(sb.GetArtifact.Do()); err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	var err error
	sb.result, err = sb.BuildManager.Build()
	return err
}

func assertArtifactNotRegistered(err error) error {
	if err == nil {
		return fmt.Errorf("artifact already registered")
	}
	if strings.Contains(err.Error(), "404 Not Found") {
		return nil
	}
	return fmt.Errorf("unable to verify artifact existence: %s", err)
}
