package cli

import (
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousQueryAds is the description of the `sous query ads` command
type SousQueryAds struct {
	Deployer     sous.Deployer
	Config       graph.LocalSousConfig
	DockerClient graph.LocalDockerClient
	StateManager *graph.ClientStateManager
	sous.Registry
	flags struct {
		singularity string
		registry    string
	}
}

func init() { QuerySubcommands["ads"] = &SousQueryAds{} }

const sousQueryAdsHelp = `The current state of deployment for every project and every cluster known to Sous.`

// Help prints the help
func (*SousQueryAds) Help() string { return sousQueryAdsHelp }

// RegisterOn adds stuff to the graph.
func (*SousQueryAds) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&config.DeployFilterFlags{})
}

// Execute defines the behavior of `sous query ads`
func (sb *SousQueryAds) Execute(args []string) cmdr.Result {
	state, err := sb.StateManager.ReadState()
	if err != nil {
		return EnsureErrorResult(err)
	}
	ads, err := sb.Deployer.RunningDeployments(sb.Registry, state.Defs.Clusters)
	if err != nil {
		return EnsureErrorResult(err)
	}
	sous.DumpDeployStatuses(os.Stdout, ads)
	return cmdr.Success()
}
