package cli

import (
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
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
	psy.Add(graph.DryrunNeither)
	psy.Add(&config.DeployFilterFlags{})
}

// Execute defines the behavior of `sous query gdm`
func (sb *SousQueryGDM) Execute(args []string) cmdr.Result {
	messages.ReportLogFieldsMessage("snapshot", logging.ExtraDebug1Level, sb.log(), sb.GDM.Snapshot())
	sous.DumpDeployments(os.Stdout, sb.GDM.Deployments)
	return cmdr.Success()
}

func (sb *SousQueryGDM) log() logging.LogSink {
	return *(logging.SilentLogSet().Child("SousQueryGDM").(*logging.LogSet))
}
