package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousUpdate is the command description for `sous update`
type SousUpdate struct {
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	OTPLFlags         config.OTPLFlags         `inject:"optional"`
	SousGraph         *graph.SousGraph
}

func init() { TopLevelCommands["update"] = &SousUpdate{} }

const sousUpdateHelp = `update the version to be deployed in a cluster

usage: sous update -cluster <name> [-tag <semver>]

sous update will update the version tag for this application in the named
cluster.
`

// Help returns the help string for this command
func (su *SousUpdate) Help() string { return sousUpdateHelp }

// AddFlags adds the flags for sous init.
func (su *SousUpdate) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &su.DeployFilterFlags, DeployFilterFlagsHelp)
}

// Execute fulfills the cmdr.Executor interface.
func (su *SousUpdate) Execute(args []string) cmdr.Result {
	update, err := su.SousGraph.GetUpdate(su.DeployFilterFlags, su.OTPLFlags)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	if err := update.Do(); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success("Updated global manifest.")
}
