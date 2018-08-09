package actions

import (
	"testing"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful/restfultest"
	"github.com/stretchr/testify/assert"
)

func TestJenkins(t *testing.T) {

	jenkins := Jenkins{}

	cl, control := restfultest.NewHTTPClientSpy()
	mani := sous.ManifestFixture("with-metadata")
	log, _ := logging.NewLogSinkSpy()

	jenkins.HTTPClient = cl
	jenkins.TargetManifestID = mani.ID()
	jenkins.Cluster = "cluster-1"
	jenkins.LogSink = log
	jenkins.User = sous.User{Name: "Fred Smith", Email: "fred@test.com"}

	updater, upctl := restfultest.NewUpdateSpy()
	control.MatchMethod(
		"Retrieve",
		spies.Once(),
		sous.ManifestFixture("with-metadata"), updater, nil,
	)
	control.Any(
		"Retrieve",
		sous.ManifestFixture("with-metadata"), restfultest.DummyUpdater(), nil,
	)
	upctl.Any(
		"Update",
		nil,
	)

	err := jenkins.Do()

	assert.NoError(t, err)

	if assert.Len(t, control.Calls(), 1) {
		args := control.Calls()[0].PassedArgs()
		assert.Regexp(t, "/manifest", args.String(0))
	}
	if assert.Len(t, upctl.Calls(), 1) {
		args := upctl.Calls()[0].PassedArgs()
		orig := sous.ManifestFixture("with-metadata")
		mani := args.Get(0).(*sous.Manifest)
		defaults := jenkins.returnJenkinsDefaultMap()
		assert.Equal(t,
			orig.Deployments["cluster-1"].Metadata["BuildBranch"],
			mani.Deployments["cluster-1"].Metadata["BuildBranch"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_DEPLOY_CI"],
			defaults["SOUS_DEPLOY_CI"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_DEPLOY_PP"],
			defaults["SOUS_DEPLOY_PP"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_DEPLOY_PROD"],
			defaults["SOUS_DEPLOY_PROD"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_INTEGRATION_TEST"],
			defaults["SOUS_INTEGRATION_TEST"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_INTEGRATION_TEST_COMMAND"],
			defaults["SOUS_INTEGRATION_TEST_COMMAND"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_SMOKE_TEST"],
			defaults["SOUS_SMOKE_TEST"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_SMOKE_TEST_COMMAND"],
			defaults["SOUS_SMOKE_TEST_COMMAND"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_STATIC_TEST"],
			defaults["SOUS_STATIC_TEST"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_STATIC_TEST_COMMAND"],
			defaults["SOUS_STATIC_TEST_COMMAND"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_UNIT_TEST"],
			defaults["SOUS_UNIT_TEST"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_UNIT_TEST_COMMAND"],
			defaults["SOUS_UNIT_TEST_COMMAND"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_USE_RC"],
			defaults["SOUS_USE_RC"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_VERSIONING_SCHEME"],
			defaults["SOUS_VERSIONING_SCHEME"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_JENKINSPIPELINE_VERSION"],
			defaults["SOUS_JENKINSPIPELINE_VERSION"],
		)
		assert.Equal(t,
			mani.Deployments["cluster-1"].Metadata["SOUS_RELEASE_BRANCH"],
			defaults["SOUS_RELEASE_BRANCH"],
		)
	}
}
