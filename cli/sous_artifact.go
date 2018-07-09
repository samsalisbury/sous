package cli

import (
	"github.com/opentable/sous/util/cmdr"
)

// SousArtifact is the `sous artifact` command
type SousArtifact struct{}

// ArtifactSubcommands collects the subcommands of `sous add` as they're added
var ArtifactSubcommands = cmdr.Commands{}

func init() { TopLevelCommands["artifact"] = &SousArtifact{} }

const sousArtifactHelp = `query and manipulate artifacts- usually not needed by users`

// Subcommands implements Submcommandor on SousArtifact
func (SousArtifact) Subcommands() cmdr.Commands {
	return ArtifactSubcommands
}

// Help implements Command for SousArtifact
func (*SousArtifact) Help() string { return sousArtifactHelp }

// Execute implements Executor on SousArtifact
func (sm *SousArtifact) Execute(args []string) cmdr.Result {
	err := cmdr.UsageErrorf("usage: sous artifact [options] <command>")
	err.Tip = "try `sous artifact help` for a list of commands"
	return err
}
