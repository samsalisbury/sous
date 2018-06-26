package cli

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousAddArtifact is the description of the `sous add artifacat` command
type SousAddArtifact struct {
	SousGraph *graph.SousGraph
}

func init() { AddSubcommands["artifact"] = &SousAddArtifact{} }

// Help prints the help
func (*SousAddArtifact) Help() string {
	return `Artifact for docker image to deploy.

This is to add the artifact to sous for deploy.
`
}

// Execute defines the behavior of `sous plumbing normalizegdm`
func (sqa *SousAddArtifact) Execute(args []string) cmdr.Result {

	/*
		plumbing, err := sqa.SousGraph.GetPlumbingNormalizeGDM()
		if err != nil {
			return cmdr.EnsureErrorResult(err)
		}

		if err := plumbing.Do(); err != nil {
			return EnsureErrorResult(err)
		}

	*/
	return cmdr.Success("Artifact added.")
}
