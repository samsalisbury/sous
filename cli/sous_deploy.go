package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousDeploy is the command description for `sous deploy`.
type SousDeploy struct {
	Config            graph.LocalSousConfig
	CLI               *CLI
	DeployFilterFlags config.DeployFilterFlags
	OTPLFlags         config.OTPLFlags
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
	MustAddFlags(fs, &sd.OTPLFlags, OtplFlagsHelp)

	fs.BoolVar(&sd.waitStable, "wait-stable", true,
		"wait for the deploy to complete before returning (otherwise, use --wait-stable=false)")
	fs.StringVar(&sd.dryrunOption, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
}

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar.
func (sd *SousDeploy) RegisterOn(psy Addable) {
	psy.Add(&sd.DeployFilterFlags)
	psy.Add(&sd.OTPLFlags)
	psy.Add(graph.DryrunOption(sd.dryrunOption))
}

// Execute fulfills the cmdr.Executor interface.
func (sd *SousDeploy) Execute(args []string) cmdr.Result {
	res := sd.CLI.Plumbing(&SousUpdate{}, []string{})

	sd.CLI.OutputResult(res)
	if !sd.CLI.IsSuccess(res) {
		return res
	}

	if sd.Config.Server != "" {
		if sd.waitStable {
			return sd.CLI.Plumbing(&SousPlumbingStatus{}, []string{})
		}
		return cmdr.Successf("Updated the global deploy manifest. Deploy in process.")
	}

	// Running serverless, so run rectify.
	rect := &SousRectify{}
	return sd.CLI.Plumbing(rect, []string{})
}
