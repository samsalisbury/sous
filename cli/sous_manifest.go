package cli

import (
	"github.com/opentable/sous/util/cmdr"
)

// SousManifest describes the `sous manifest` command
type SousManifest struct{}

// ManifestSubcommands holds the subcommands of `sous manifest`
var ManifestSubcommands = cmdr.Commands{}

func init() { TopLevelCommands["manifest"] = &SousManifest{} }

const sousManifestHelp = `query and manipulate deployment manifests`

// Subcommands implements Subcommander on SousManifest
func (SousManifest) Subcommands() cmdr.Commands {
	return ManifestSubcommands
}

// Help implements Command on SousManifest
func (*SousManifest) Help() string { return sousManifestHelp }

// Execute implements Executor on SousManifest
func (sm *SousManifest) Execute(args []string) cmdr.Result {
	err := cmdr.UsageErrorf("usage: sous manifest [options] <command>")
	err.Tip = "try `sous manifest help` for a list of commands"
	return err
}
