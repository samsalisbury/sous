package cli

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousAdd is the `sous add` subcommand
type SousGet struct{}

// GetSubcommands collects the subcommands of `sous add` as they're added
var GetSubcommands = cmdr.Commands{}

func init() { TopLevelCommands["get"] = &SousGet{} }

const sousGetHelp = `Get subcommands of Sous - usually not needed by users`

// RegisterOn implements Registrant on SousAdd
func (SousGet) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
}

// Subcommands implements Submcommandor for SousPlumbing
func (SousGet) Subcommands() cmdr.Commands {
	return GetSubcommands
}

// Help implements Command for SousAdd
func (*SousGet) Help() string { return sousGetHelp }
