package cli

import (
	"encoding/json"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousContext is the 'sous context' command.
type SousContext struct {
	SourceContext func() (*sous.SourceContext, error)
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
	sc, err := sv.SourceContext()
	if err != nil {
		return EnsureErrorResult(err)
	}
	b, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return EnsureErrorResult(err)
	}
	return Successf(string(b))
}
