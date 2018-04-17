package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousNewDeploy has the same interface as SousDeploy, but uses the new
// PUT /single-deployment endpoint to begin the deployment, and polls by
// watching the returned rectification URL.
type SousNewDeploy struct {
	SousGraph *graph.SousGraph

	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	waitStable        bool
	force             bool
	dryrunOption      string
}

func init() { TopLevelCommands["newdeploy"] = &SousNewDeploy{} }

const sousNewDeployHelp = `deploys a new version into a particular cluster

usage: sous newdeploy [(options)]

sous newdeploy will deploy the version tag for this application in the named
cluster.

DEPRECATED: This now does the same thing as sous deploy, and this alias will be
removed in the future. Please update your scripts, documentation and habits
accordingly.`

// Help returns the help string for this command.
func (sd *SousNewDeploy) Help() string { return sousNewDeployHelp }

// AddFlags adds the flags for sous init.
func (sd *SousNewDeploy) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sd.DeployFilterFlags, NewDeployFilterFlagsHelp)

	fs.BoolVar(&sd.force, "force", false,
		"force deploy no matter if GDM already is at the correct version")
	fs.BoolVar(&sd.waitStable, "wait-stable", true,
		"wait for the deploy to complete before returning (otherwise, use --wait-stable=false)")
	fs.StringVar(&sd.dryrunOption, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
}

// Execute creates the new deployment.
func (sd *SousNewDeploy) Execute(args []string) cmdr.Result {
	deploy, err := sd.SousGraph.GetDeploy(sd.DeployFilterFlags, sd.dryrunOption, sd.force, sd.waitStable)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := deploy.Do(); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success("Done.")
}
