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

const sousQueryGDMHelp = `The intended state of deployment for every project and every cluster known to Sous.

The results of 'sous query gdm' and 'sous query ads' will not be identical if
a problem is preventing sous from modifying the current state of Singularity.
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
	return cmdr.Success()
}
