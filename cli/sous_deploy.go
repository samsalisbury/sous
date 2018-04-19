package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousDeploy is the command description for `sous deploy`.
type SousDeploy struct {
	SousGraph *graph.SousGraph

	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	waitStable        bool
	force             bool
	dryrunOption      string
}

func init() { TopLevelCommands["deploy"] = &SousDeploy{} }

const sousDeployHelp = `deploys a new version into a particular cluster

usage: sous deploy (options)

sous deploy will deploy the version tag for this application in the named
cluster.
`

// Help returns the help string for this command.
func (sd *SousDeploy) Help() string { return sousDeployHelp }

// AddFlags adds the flags for sous init.
func (sd *SousDeploy) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sd.DeployFilterFlags, NewDeployFilterFlagsHelp)

	fs.BoolVar(&sd.force, "force", false,
		"force deploy no matter if GDM already is at the correct version")
	fs.BoolVar(&sd.waitStable, "wait-stable", true,
		"wait for the deploy to complete before returning (otherwise, use --wait-stable=false)")
	fs.StringVar(&sd.dryrunOption, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
}

// Execute fulfills the cmdr.Executor interface.
func (sd *SousDeploy) Execute(args []string) cmdr.Result {
	deploy, err := sd.SousGraph.GetDeploy(sd.DeployFilterFlags, sd.dryrunOption, sd.force, sd.waitStable)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := deploy.Do(); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success("Done.")
}
