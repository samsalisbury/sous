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
	Config       LocalSousConfig
	DockerClient LocalDockerClient
	Sous         *Sous
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
	if len(args) < 1 {
		return UsageErrorf("sous querry adc: directory to load deployment configuration required")
	}
	dir := args[0]

	state, err := sous.LoadState(dir)
	if err != nil {
		return EnsureErrorResult(err)
	}

	nc := sous.NewNameCache(
		sb.DockerClient,
		sb.Config.DatabaseDriver,
		sb.Config.DatabaseConnection)
	ra := sous.NewRectiAgent(nc)
	sc := sous.NewSetCollector(ra)
	ads, err := sc.GetRunningDeployment(state.BaseURLs())
	if err != nil {
		return EnsureErrorResult(err)
	}

	w := &tabwriter.Writer{}
	w.Init(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, sous.TabbedDeploymentHeaders())

	for _, d := range ads {
		fmt.Fprintln(w, d.Tabbed())
	}
	w.Flush()

	return Success()
}
