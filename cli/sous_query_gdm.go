package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousQueryGDM is the description of the `sous query gdm` command
type SousQueryGDM struct {
	Sous  *Sous
	flags struct {
		singularity string
		registry    string
	}
}

func init() { QuerySubcommands["gdm"] = &SousQueryGDM{} }

const sousQueryGDMHelp = `
Loads the current deployment configuration and prints it out

This should resemble the manifest that was used to establish the intended state of deployment.
`

// Help prints the help
func (*SousQueryGDM) Help() string { return sousQueryGDMHelp }

// Execute defines the behavior of `sous query gdm`
func (sb *SousQueryGDM) Execute(args []string) cmdr.Result {
	if len(args) < 1 {
		return UsageErrorf("sous querry gdm: directory to load deployment configuration required")
	}
	dir := args[0]

	state, err := sous.LoadState(dir)
	if err != nil {
		return EnsureErrorResult(err)
	}

	gdm, err := state.Deployments()
	if err != nil {
		return EnsureErrorResult(err)
	}

	w := &tabwriter.Writer{}
	w.Init(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, sous.TabbedDeploymentHeaders())

	for _, d := range gdm {
		fmt.Fprintln(w, d.Tabbed())
	}
	w.Flush()

	return Success()
}
