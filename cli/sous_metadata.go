package cli

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousMetadata describes the `sous metadata` command.
type SousMetadata struct{}

// MetadataSubcommands holds the subcommands of `sous metadata`.
var MetadataSubcommands = cmdr.Commands{}

func init() { TopLevelCommands["metadata"] = &SousMetadata{} }

const sousMetadataHelp = `query and manipulate deployment metadata

The "sous metadata" command is an alternate means of configuring and
viewing the subset of "metadata" values that can also be manipulated
by the "sous manifest" command.

Metadata values do not change any behaviors in Sous. They exist to store
data needed for software that depends on the Sous server API, such as
build systems.
`

// Subcommands implements Subcommander on SousMetadata.
func (SousMetadata) Subcommands() cmdr.Commands {
	return MetadataSubcommands
}

// RegisterOn implements Registrant on SousMetadata
func (SousMetadata) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
}

// Help implements Command on SousMetadata.
func (*SousMetadata) Help() string { return sousMetadataHelp }

// Execute implements Executor on SousMetadata.
func (sm *SousMetadata) Execute(args []string) cmdr.Result {
	err := cmdr.UsageErrorf("usage: sous metadata [options] <command>")
	err.Tip = "try `sous help metadata` for a list of commands"
	return err
}
