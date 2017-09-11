package cli

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousPlumbing is the `sous plumbing` subcommand
type SousPlumbing struct{}

// PlumbingSubcommands collects the subcommands of `sous plumbing` as they're added
var PlumbingSubcommands = cmdr.Commands{}

func init() { TopLevelCommands["plumbing"] = &SousPlumbing{} }

const sousPlumbingHelp = `Plumbing subcommands of Sous - usually not needed by users`

// RegisterOn implements Registrant on SousPlumbing
func (SousPlumbing) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
}

// Subcommands implements Submcommandor for SousPlumbing
func (SousPlumbing) Subcommands() cmdr.Commands {
	return PlumbingSubcommands
}

// Help implements Command for SousPlumbing
func (*SousPlumbing) Help() string { return sousPlumbingHelp }
