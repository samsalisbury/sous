package cli

import (
	"encoding/json"

	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/cmdr"
)

type SousContext struct {
	Context sous.BuildContext
}

func init() { TopLevelCommands["version"] = &SousVersion{} }

const sousContextHelp = `
show the current build context

context prints out sous's view of your current context

args:
`

func (*SousContext) Help() string { return sousVersionHelp }

func (sv *SousContext) Execute(args []string) cmdr.Result {
	b, err := json.MarshalIndent(sv.Context, "", "  ")
	if err != nil {
		return EnsureErrorResult(err)
	}
	return Successf(string(b))
}
