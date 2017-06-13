package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/pkg/errors"
)

type SousMetadataSet struct {
	config.DeployFilterFlags
	*sous.ResolveFilter
	*sous.State
	graph.CurrentGDM
	sous.StateWriter
	WriteContext graph.StateWriteContext
}

func init() { MetadataSubcommands["set"] = &SousMetadataSet{} }

const sousMetadataSetHelp = `set deployment metadata`

func (*SousMetadataSet) Help() string { return sousMetadataSetHelp }

func (smg *SousMetadataSet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, MetadataFilterFlagsHelp)
}

func (smg *SousMetadataSet) RegisterOn(psy Addable) {
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousMetadataSet) Execute(args []string) cmdr.Result {
	if len(args) < 2 {
		return EnsureErrorResult(errors.Errorf("<name> and <value> both required"))
	}
	if smg.DeployFilterFlags.Repo == "" {
		return EnsureErrorResult(errors.Errorf("-repo is required"))
	}

	key := args[0]
	value := args[1]

	filtered := smg.CurrentGDM.Clone().Filter(smg.ResolveFilter.FilterDeployment)
	insertion := make([]*sous.Deployment, 0, filtered.Len())
	for _, dep := range filtered.Snapshot() {
		dep.Metadata[key] = value
		insertion = append(insertion, dep)
	}

	if err := smg.State.UpdateDeployments(insertion...); err != nil {
		return EnsureErrorResult(err)
	}

	ctx := sous.StateContext(smg.WriteContext)
	if err := smg.StateWriter.WriteState(smg.State, ctx); err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}
