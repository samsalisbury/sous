package cli

import (
	"github.com/opentable/sous/util/cmdr"
)

// SousQuery wraps subcommands.
type SousQuery struct{}

// QuerySubcommands are the subcommands of 'sous query'.
var QuerySubcommands = cmdr.Commands{}

func init() { TopLevelCommands["query"] = &SousQuery{} }

const sousQueryHelp = `inquire about sous details`

// Help returns help for 'sous query'
func (*SousQuery) Help() string { return sousQueryHelp }

// Subcommands returns the subcommands of 'sous query'.
func (SousQuery) Subcommands() cmdr.Commands {
	return QuerySubcommands
}

// Execute prints help, as 'sous query' requires a subcommand.
func (sb *SousQuery) Execute(args []string) cmdr.Result {
	err := cmdr.UsageErrorf("usage: sous query [options] <command>")
	err.Tip = "try `sous help query` for a list of commands"
	return err
}
