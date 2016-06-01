package cli

import (
	"encoding/json"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

type SousContext struct {
	SourceContext *sous.SourceContext
}

func init() { TopLevelCommands["context"] = &SousContext{} }

const sousContextHelp = `
show the current build context

context prints out sous's view of your current context

args:
`

func (*SousContext) Help() string { return sousContextHelp }

func (sv *SousContext) Execute(args []string) cmdr.Result {
	b, err := json.MarshalIndent(sv.SourceContext, "", "  ")
	if err != nil {
		return EnsureErrorResult(err)
	}
	return Successf(string(b))
}
