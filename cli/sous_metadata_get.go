package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/pkg/errors"
)

type SousMetadataGet struct {
	config.DeployFilterFlags
	*sous.ResolveFilter
	*sous.State
	graph.CurrentGDM
}

func init() { MetadataSubcommands["get"] = &SousMetadataGet{} }

const sousMetadataGetHelp = `
query deployment metadata
`

func (*SousMetadataGet) Help() string { return sousMetadataHelp }

func (smg *SousMetadataGet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, MetadataFilterFlagsHelp)
}

func (smg *SousMetadataGet) RegisterOn(psy Addable) {
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousMetadataGet) Execute(args []string) cmdr.Result {
	if smg.DeployFilterFlags.Repo == "" {
		return EnsureErrorResult(errors.Errorf("-repo is required"))
	}
	if smg.DeployFilterFlags.Offset == "" {
		return EnsureErrorResult(errors.Errorf("-offset is required"))
	}
	filtered := smg.CurrentGDM.Clone().Filter(smg.ResolveFilter.FilterDeployment)
	manis, err := filtered.Manifests(smg.State.Defs)
	if err != nil {
		return EnsureErrorResult(err)
	}
	if manis.Len() > 1 {
		panic("Filtered manifests contained more than one entry")
	}

	return Success()
}
