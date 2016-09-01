package cli

import (
	"os"

	"github.com/opentable/sous/util/cmdr"
)

type SousHelp struct {
	CLI  *CLI
	Sous *Sous
}

func init() { TopLevelCommands["help"] = &SousHelp{} }

const sousHelpHelp = `
get help with sous

help shows help information for sous itself, as well as all its subcommands
for detailed help with any command, use 'sous help <command>'.

args: [command]
`

func (sh *SousHelp) Help() string { return sousHelpHelp }

func (sh *SousHelp) Execute(args []string) cmdr.Result {
	// Get the name this instance was invoked with.
	name := os.Args[0]
	help, err := sh.CLI.Help(sh.Sous, name, args)
	if err != nil {
		return EnsureErrorResult(err)
	}
	return Successf(help)
}
