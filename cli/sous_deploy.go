package cli

import (
	"flag"
	"fmt"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousDeploy is the command description for `sous deploy`.
type SousDeploy struct {
	SousGraph *graph.SousGraph

	Config            graph.LocalSousConfig
	CLI               *CLI
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	OTPLFlags         config.OTPLFlags         `inject:"optional"`
	dryrunOption      string
	waitStable        bool
}

func init() { TopLevelCommands["deploy"] = &SousDeploy{} }

const sousDeployHelp = `deploys a new version into a particular cluster

usage: sous deploy -cluster <name> -tag <semver> [-use-otpl-deploy|-ignore-otpl-deploy]

sous deploy will deploy the version tag for this application in the named
cluster.
`

// Help returns the help string for this command.
func (sd *SousDeploy) Help() string { return sousDeployHelp }

// AddFlags adds the flags for sous init.
func (sd *SousDeploy) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sd.DeployFilterFlags, DeployFilterFlagsHelp)
	//MustAddFlags(fs, &sd.OTPLFlags, OtplFlagsHelp)

	fs.BoolVar(&sd.waitStable, "wait-stable", true,
		"wait for the deploy to complete before returning (otherwise, use --wait-stable=false)")
	fs.StringVar(&sd.dryrunOption, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
}

// Execute fulfills the cmdr.Executor interface.
func (sd *SousDeploy) Execute(args []string) cmdr.Result {
	//func GetUpdate(di injector, dff config.DeployFilterFlags, otpl config.OTPLFlags) Action {
	update, err := sd.SousGraph.GetUpdate(sd.DeployFilterFlags, sd.OTPLFlags)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	if err := update.Do(); err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	// Running serverless, so run rectify.
	if sd.Config.Server == "" {
		rectify, err := sd.SousGraph.GetRectify(sd.dryrunOption, sd.DeployFilterFlags)
		if err != nil {
			return cmdr.EnsureErrorResult(err)
		}
		if err := rectify.Do(); err != nil {
			return cmdr.EnsureErrorResult(err)
		}
		return cmdr.Success("Successfully rectified")
	}

	if sd.waitStable {
		fmt.Fprintf(sd.CLI.Out, "Waiting for server to report that deploy has stabilized...\n")

		poll, err := sd.SousGraph.GetPollStatus(sd.dryrunOption, sd.DeployFilterFlags)
		if err != nil {
			return cmdr.EnsureErrorResult(err)
		}
		if err := poll.Do(); err != nil {
			return cmdr.EnsureErrorResult(err)
		}
		return cmdr.Success("")
	}
	return cmdr.Successf("Deploy in process.")

}
