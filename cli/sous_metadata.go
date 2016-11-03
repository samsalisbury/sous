package cli

import "github.com/opentable/sous/util/cmdr"

type SousMetadata struct{}

var MetadataSubcommands = cmdr.Commands{}

func init() { TopLevelCommands["query"] = &SousMetadata{} }

const sousMetadataHelp = `
query and manipulate deployment metadata
`

func (SousMetadata) Subcommands() cmdr.Commands {
	return MetadataSubcommands
}

func (*SousMetadata) Help() string { return sousMetadataHelp }

func (sm *SousMetadata) Execute(args []string) cmdr.Result {
	err := UsageErrorf("usage: sous metadata [options] <command>")
	err.Tip = "try `sous metadata help` for a list of commands"
	return err
}
