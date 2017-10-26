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
	graph.TargetManifestID
	graph.HTTPClient

	User sous.User
}

func init() { MetadataSubcommands["set"] = &SousMetadataSet{} }

const sousMetadataSetHelp = `set deployment metadata`

func (*SousMetadataSet) Help() string { return sousMetadataSetHelp }

func (smg *SousMetadataSet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, MetadataFilterFlagsHelp)
}

func (smg *SousMetadataSet) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousMetadataSet) Execute(args []string) cmdr.Result {
	if len(args) < 2 {
		return EnsureErrorResult(errors.Errorf("<name> and <value> both required"))
	}
	key := args[0]
	value := args[1]

	mani := sous.Manifest{}
	up, err := smg.HTTPClient.Retrieve("/manifest", smg.TargetManifestID.QueryMap(), &mani, nil)
	if err != nil {
		return EnsureErrorResult(err)
	}

	for cname, depspec := range mani.Deployments {
		if !smg.ResolveFilter.FilterClusterName(cname) {
			continue
		}
		depspec.Metadata[key] = value
	}

	if err := up.Update(&mani, smg.User.HTTPHeaders()); err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}
