package cli

import (
	"testing"

	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func TestSlackMessage(t *testing.T) {

	sd := &SousDeploy{}
	user := sous.User{
		Name:  "Bob Smith",
		Email: "bsmith@test.com",
	}

	config := &config.Config{
		User: user,
	}

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
		User:               user,
		Config:             config,
	}
	var dAction actions.Action
	dAction = d
	assert.NotPanics(t, func() { sd.slackMessage(dAction, nil) }, "shouldn't panic")

}
