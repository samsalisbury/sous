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
	GDM   CurrentGDM
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
	sous.Log.Vomit.Printf("%v", sb.GDM.Snapshot())
	w := &tabwriter.Writer{}
	w.Init(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, sous.TabbedDeploymentHeaders())

	for _, d := range sb.GDM.Snapshot() {
		fmt.Fprintln(w, d.Tabbed())
	}
	w.Flush()

	return Success()
}
