package cli

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

type SousMetadata struct{}

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

func (SousMetadata) Subcommands() cmdr.Commands {
	return MetadataSubcommands
}

func (SousMetadata) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
}

func (*SousMetadata) Help() string { return sousMetadataHelp }

func (sm *SousMetadata) Execute(args []string) cmdr.Result {
	err := cmdr.UsageErrorf("usage: sous metadata [options] <command>")
	err.Tip = "try `sous help metadata` for a list of commands"
	return err
}
