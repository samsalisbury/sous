package cli

import (
	"flag"

	slack "github.com/ashwanthkumar/slack-go-webhook"
	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
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
		sd.slackMessage(deploy, err)
		return cmdr.EnsureErrorResult(err)
	}

	err = deploy.Do()

	sd.slackMessage(deploy, err)

	if err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success("Done.")
}

func (sd *SousDeploy) slackMessage(action actions.Action, err error) {

	var slackURL, slackChannel string
	d, ok := action.(*actions.Deploy)

	if ok {
		slackURL = d.Config.SlackHookURL
		slackChannel = d.Config.SlackChannel
	}

	if len(slackURL) < 1 || len(slackChannel) < 1 {
		return
	}

	version, _ := d.ResolveFilter.TagVersion()

	messages.ReportLogFieldsMessage("SlackMessage", logging.DebugLevel, d.LogSink, d.TargetDeploymentID.ManifestID, version)

	color := "good"
	attachment := slack.Attachment{Color: &color}
	attachment.AddField(slack.Field{Title: "Build Author", Value: d.Config.User.Name})
	attachment.AddField(slack.Field{Title: "Manifest ID", Value: d.TargetDeploymentID.String()})
	attachment.AddField(slack.Field{Title: "Version", Value: version.String()})

	if err != nil {
		attachment.AddField(slack.Field{Title: "Status", Value: "FAILED"}).AddField(slack.Field{Title: "Error", Value: err.Error()})
		color = "danger"
	} else {
		attachment.AddField(slack.Field{Title: "Status", Value: "SUCCESS"})
	}

	payload := slack.Payload{
		Username:    "Sous Bot",
		Channel:     slackChannel,
		IconEmoji:   ":chefhat:",
		Attachments: []slack.Attachment{attachment},
	}

	errs := slack.Send(slackURL, "", payload)

	if len(errs) > 0 {
		messages.ReportLogFieldsMessage("Error sending slack message", logging.WarningLevel, d.LogSink, errs)
	}

}
