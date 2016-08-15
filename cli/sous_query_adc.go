package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousQueryAdc is the description of the `sous query adc` command
type SousQueryAdc struct {
	Deployer     sous.Deployer
	Config       LocalSousConfig
	DockerClient LocalDockerClient
	GDM          CurrentGDM
	State        *sous.State
	flags        struct {
		singularity string
		registry    string
	}
}

func init() { QuerySubcommands["adc"] = &SousQueryAdc{} }

const sousQueryAdcHelp = `
Queries the Singularity server and container registry to determine a synthetic global manifest.

This should resemble the manifest that was used to establish the current state of deployment.
`

// Help prints the help
func (*SousQueryAdc) Help() string { return sousBuildHelp }

// Execute defines the behavior of `sous query adc`
func (sb *SousQueryAdc) Execute(args []string) cmdr.Result {
	ads, err := sb.Deployer.RunningDeployments(sb.State.Defs.Clusters)
	if err != nil {
		return EnsureErrorResult(err)
	}

	w := &tabwriter.Writer{}
	w.Init(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, sous.TabbedDeploymentHeaders())

	for _, d := range ads.Snapshot() {
		fmt.Fprintln(w, d.Tabbed())
	}
	w.Flush()

	return Success()
}
