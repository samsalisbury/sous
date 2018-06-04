package cli

import (
	"testing"

	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func returnAction(t *testing.T, c *config.Config) actions.Action {
	user := sous.User{
		Name:  "Bob Smith",
		Email: "bsmith@test.com",
	}

	c.User = user
	rMid := sous.MustParseManifestID("github.com/blah/blah")
	rf := &sous.ResolveFilter{
		Repo:    sous.NewResolveFieldMatcher("github.com/from/flags"),
		Tag:     sous.NewResolveFieldMatcher("1.1.1"),
		Cluster: sous.NewResolveFieldMatcher("dev-ci"),
	}
	did, err := rf.DeploymentID(rMid)
	assert.NoError(t, err)
	ls, _ := logging.NewLogSinkSpy()
	d := &actions.Deploy{
		ResolveFilter:      rf,
		TargetDeploymentID: did,
		LogSink:            ls,
		User:               c.User,
		Config:             c,
	}
	return d
}

func TestSlackMessage_just_hookandchannel(t *testing.T) {
	sd := &SousDeploy{}
	config := &config.Config{
		SlackHookURL: "http://foo.com",
		SlackChannel: "bar",
	}

	dAction := returnAction(t, config)
	assert.NotPanics(t, func() { sd.slackMessage(dAction, nil) }, "shouldn't panic")

}

func TestSlackMessage_noslack(t *testing.T) {
	sd := &SousDeploy{}
	config := &config.Config{}

	dAction := returnAction(t, config)
	assert.NotPanics(t, func() { sd.slackMessage(dAction, nil) }, "shouldn't panic")

}

func TestSlackMessage_justadditinonal(t *testing.T) {
	additionalChannels := make(map[string]string)
	sd := &SousDeploy{}

	additionalChannels["bar"] = "http://foo.com"

	config := &config.Config{
		AdditionalSlackChannels: additionalChannels,
	}

	dAction := returnAction(t, config)
	assert.NotPanics(t, func() { sd.slackMessage(dAction, nil) }, "shouldn't panic")

}

func TestSlackMessage_both(t *testing.T) {
	additionalChannels := make(map[string]string)
	sd := &SousDeploy{}

	additionalChannels["channel"] = "http://hook.com"

	config := &config.Config{
		SlackHookURL:            "http://foo.com",
		SlackChannel:            "bar",
		AdditionalSlackChannels: additionalChannels,
	}

	dAction := returnAction(t, config)
	assert.NotPanics(t, func() { sd.slackMessage(dAction, nil) }, "shouldn't panic")

}
