package cli

import (
	"flag"
	"os"

	slack "github.com/ashwanthkumar/slack-go-webhook"
	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
)

// SousSingReqID is the command description for `sous deploy`.
type SousSingReqID struct {
	SousGraph *graph.SousGraph
	DFF       config.DeployFilterFlags `inject:"optional"`
}

func init() { TopLevelCommands["singreqid"] = &SousSingReqID{} }

const sousSingReqIDHelp = `returns the Singularity request ID of a deployment

usage: sous singreqid -repo <repo> -cluster <cluster> [-offset <offset>] [-flavor <flavor>]
`

// Help returns the help string for this command.
func (sd *SousSingReqID) Help() string { return sousDeployHelp }

// AddFlags adds the flags for sous init.
func (sd *SousSingReqID) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sd.DFF, NewDeployFilterFlagsHelp)
}

// Execute fulfills the cmdr.Executor interface.
func (sd *SousSingReqID) Execute(args []string) cmdr.Result {

	var up restful.Updater
	mg, err := sd.SousGraph.GetManifestGet(sd.DFF, os.Stdout, &up)
	if err != nil {
		return EnsureErrorResult(err)
	}
	m, err := mg.GetManifest()
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	d, ok := m.Deployments[sd.DFF.Cluster]
	if !ok {
		return cmdr.UsageErrorf("manifest %q does not have a deployment for %q",
			m.ID(), sd.DFF.Cluster)
	}

	if d.SingularityRequestID != "" {
		return cmdr.Success(d.SingularityRequestID)
	}

	did, err := sd.DFF.DeploymentIDFlags.DeploymentID()
	if err != nil {
		return cmdr.UsageErrorf("invalid flags: %s", err)
	}
	computed, err := singularity.MakeRequestID(did)
	if err != nil {
		return cmdr.UsageErrorf("computing request ID: %s", err)
	}

	return cmdr.Success(computed)
}

func (sd *SousSingReqID) slackMessage(action actions.Action, err error) {

	var slackURL, slackChannel string
	var additionalChannels map[string]string

	d, ok := action.(*actions.Deploy)

	if ok {
		slackURL = d.Config.SlackHookURL
		slackChannel = d.Config.SlackChannel
		additionalChannels = d.Config.AdditionalSlackChannels
	}

	if additionalChannels == nil {
		additionalChannels = make(map[string]string)
	}

	if len(slackURL) > 0 && len(slackChannel) > 0 {
		additionalChannels[slackChannel] = slackURL
	}

	if len(additionalChannels) < 1 {
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
		IconEmoji:   ":chefhat:",
		Attachments: []slack.Attachment{attachment},
	}

	var errs []error
	for k, v := range additionalChannels {
		payload.Channel = k
		newErrors := slack.Send(v, "", payload)
		errs = append(errs, newErrors...)
	}

	if len(errs) > 0 {
		messages.ReportLogFieldsMessage("Error sending slack message", logging.WarningLevel, d.LogSink, errs)
	}

}
