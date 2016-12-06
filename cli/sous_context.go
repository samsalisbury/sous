package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousContext is the 'sous context' command.
type SousContext struct {
	config.DeployFilterFlags
	*sous.SourceContext
}

func init() { TopLevelCommands["context"] = &SousContext{} }

const sousContextHelp = `show the current build context

context prints out sous's view of your current context

args:
`

// Help provides help for sous context.
func (*SousContext) Help() string { return sousContextHelp }

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar
func (sc *SousContext) RegisterOn(psy Addable) {
	psy.Add(&sc.DeployFilterFlags)
}

// AddFlags adds flags to the context command.
func (sc *SousContext) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sc.DeployFilterFlags, SourceFlagsHelp)
	//fs.BoolVar(&sb.PolicyFlags.ForceClone, "force-clone", false, "force a shallow clone of the codebase before build")
	// above is commented prior to impl.
}

// Execute prints the detected sous context.
func (sc *SousContext) Execute(args []string) cmdr.Result {
	return SuccessYAML(sc.SourceContext)
}
