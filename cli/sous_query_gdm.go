package cli

import (
	"os"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousQueryGDM is the description of the `sous query gdm` command
type SousQueryGDM struct {
	GDM   graph.CurrentGDM
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

func (*SousQueryGDM) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunOption("none"))
}

// Execute defines the behavior of `sous query gdm`
func (sb *SousQueryGDM) Execute(args []string) cmdr.Result {
	sous.Log.Vomit.Printf("%v", sb.GDM.Snapshot())
	sous.DumpDeployments(os.Stdout, sb.GDM.Deployments)
	return Success()
}
