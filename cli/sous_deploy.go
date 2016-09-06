package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/util/cmdr"
)

// SousDeploy is the command description for `sous deploy`
type SousDeploy struct {
	*CLI
	config.DeployFilterFlags
	config.OTPLFlags
	Update       SousUpdate
	Rectify      SousRectify
	rectifyFlags struct {
		dryrun string
	}
}

func init() { TopLevelCommands["deploy"] = &SousDeploy{} }

const sousDeployHelp = `
deploys a new version into a particular cluster

usage: sous deploy -cluster <name> -tag <semver> [-use-otpl-deploy|-ignore-otpl-deploy]

sous deploy will deploy the version tag for this application in the named
cluster.
`

// Help returns the help string for this command
func (sd *SousDeploy) Help() string { return sousDeployHelp }

// AddFlags adds the flags for sous init.
func (sd *SousDeploy) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sd.DeployFilterFlags, config.DeployFilterFlagsHelp)
	AddFlags(fs, &sd.OTPLFlags, config.OtplFlagsHelp)

	fs.StringVar(&sd.rectifyFlags.dryrun, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
}

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar
func (sd *SousDeploy) RegisterOn(psy Addable) {
	psy.Add(&sd.DeployFilterFlags)
	psy.Add(&sd.OTPLFlags)
}

// Execute fulfills the cmdr.Executor interface.
func (sd *SousDeploy) Execute(args []string) cmdr.Result {
	res := sd.CLI.Plumbing(&SousUpdate{
		DeployFilterFlags: sd.DeployFilterFlags,
		OTPLFlags:         sd.OTPLFlags,
	}, []string{})
	if !sd.CLI.IsSuccess(res) {
		return res
	}
	sd.CLI.OutputResult(res)

	rect := &SousRectify{SourceFlags: sd.DeployFilterFlags}
	rect.flags.dryrun = sd.rectifyFlags.dryrun

	return sd.CLI.Plumbing(rect, []string{})
}
