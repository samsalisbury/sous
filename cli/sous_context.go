package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousContext is the 'sous context' command.
type SousContext struct {
	*sous.SourceContext
}

func init() { TopLevelCommands["context"] = &SousContext{} }

const sousContextHelp = `
show the current build context

context prints out sous's view of your current context

args:
`

// Help provides help for sous context.
func (*SousContext) Help() string { return sousContextHelp }

// Execute prints the detected sous context.
func (sv *SousContext) Execute(args []string) cmdr.Result {
	return SuccessYAML(sv.SourceContext)
}
