package cli

import "github.com/opentable/sous/util/cmdr"

type SousManifest struct{}

var ManifestSubcommands = cmdr.Commands{}

func init() { TopLevelCommands["manifest"] = &SousManifest{} }

const sousManifestHelp = `query and manipulate deployment manifests`

func (SousManifest) Subcommands() cmdr.Commands {
	return ManifestSubcommands
}

func (*SousManifest) Help() string { return sousManifestHelp }

func (sm *SousManifest) Execute(args []string) cmdr.Result {
	err := UsageErrorf("usage: sous manifest [options] <command>")
	err.Tip = "try `sous manifest help` for a list of commands"
	return err
}
