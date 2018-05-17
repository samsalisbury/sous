package cli

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousPlumbingNormalizeGDM is the description of the `sous plumbing normalizegdm` command
type SousPlumbingNormalizeGDM struct {
	SousGraph *graph.SousGraph
	graph.LocalSousConfig
}

func init() { PlumbingSubcommands["normalizegdm"] = &SousPlumbingNormalizeGDM{} }

// Help prints the help
func (*SousPlumbingNormalizeGDM) Help() string {
	return `Normalizes the storage format of GDM.

Loads and saves the GDM, such that it's storage format will be normalized.
This is needed sometimes after manually editing the GDM, so that spurious
formatting changes won't be considered real, conflicting updates.
`
}

// Execute defines the behavior of `sous plumbing normalizegdm`
func (sqa *SousPlumbingNormalizeGDM) Execute(args []string) cmdr.Result {

	plumbing, err := sqa.SousGraph.GetPlumbingNormalizeGDM(sqa.LocalSousConfig.StateLocation)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := plumbing.Do(); err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success("Normalized.")
}
