package cli

import (
	"os"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousQueryAds is the description of the `sous query ads` command
type SousQueryAds struct {
	Deployer     sous.Deployer
	Config       graph.LocalSousConfig
	DockerClient graph.LocalDockerClient
	GDM          graph.CurrentGDM
	State        *sous.State
	flags        struct {
		singularity string
		registry    string
	}
}

func init() { QuerySubcommands["ads"] = &SousQueryAds{} }

const sousQueryAdsHelp = `
Queries the Singularity server and container registry to determine a synthetic global manifest.

This should resemble the manifest that was used to establish the current state of deployment.
`

// Help prints the help
func (*SousQueryAds) Help() string { return sousBuildHelp }

// Execute defines the behavior of `sous query ads`
func (sb *SousQueryAds) Execute(args []string) cmdr.Result {
	ads, err := sb.Deployer.RunningDeployments(sb.State.Defs.Clusters)
	if err != nil {
		return EnsureErrorResult(err)
	}
	sous.DumpDeployments(os.Stdout, ads)
	return Success()
}
