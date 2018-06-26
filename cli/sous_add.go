package cli

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousAdd is the `sous add` subcommand
type SousAdd struct{}

// AddSubcommands collects the subcommands of `sous add` as they're added
var AddSubcommands = cmdr.Commands{}

func init() { TopLevelCommands["add"] = &SousAdd{} }

const sousAddHelp = `Add subcommands of Sous - usually not needed by users`

// RegisterOn implements Registrant on SousAdd
func (SousAdd) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
}

// Subcommands implements Submcommandor for SousPlumbing
func (SousAdd) Subcommands() cmdr.Commands {
	return AddSubcommands
}

// Help implements Command for SousAdd
func (*SousAdd) Help() string { return sousAddHelp }
