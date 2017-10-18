package graph

import (
	"testing"

	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/docker"
	sous "github.com/opentable/sous/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureDeployFilterFlags() config.DeployFilterFlags {
	return config.DeployFilterFlags{
		Repo:    "github.com/somewhere",
		Offset:  "",
		Flavor:  "vanilla",
		Tag:     "0.1.22",
		Cluster: "test-cluster",
		All:     false,
	}
}

func fixtureGraph() *SousGraph {
	return &SousGraph{}
}

func TestActionUpdate(t *testing.T) {
	action := fixtureGraph().GetUpdate(fixtureDeployFilterFlags(), config.OTPLFlags{})
	update, rightType := action.(*actions.Update)
	require.True(t, rightType)

	assert.Equal(t, "github.com/example/project", update.ResolveFilter.Repo.ValueOr("{globbed!!!}"))
	assert.Equal(t, "github.com/example/project", update.Manifest.ID().Source.Repo)
}

func TestActionPollStatus(t *testing.T) {
	action := fixtureGraph().GetPollStatus(fixtureDeployFilterFlags())
	pollStatus, rightType := action.(*actions.PollStatus)
	require.True(t, rightType)

	require.NotNil(t, pollStatus.StatusPoller)
	assert.NotEqual(t, "", pollStatus.StatusPoller.Repo)
	assert.Equal(t, pollStatus.StatusPoller.ResolveFilter.Repo.ValueOr("{globbed!!!}"), fixtureDeployFilterFlags().Repo)
}

func TestActionRectify(t *testing.T) {
	action := fixtureGraph().GetRectify("none", fixtureDeployFilterFlags())

	rect, rightType := action.(*actions.Rectify)
	require.True(t, rightType)

	assert.NotNil(t, rect.State)
	require.NotNil(t, rect.Resolver)
	require.NotNil(t, rect.Resolver.ResolveFilter)
	assert.Equal(t, rect.Resolver.ResolveFilter.All(), false)
}

func TestActionRectifyDryruns(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testDryRun := func(which string) (sous.Deployer, sous.Registry) {
		action := fixtureGraph().GetRectify("none", fixtureDeployFilterFlags())
		require.IsType(&actions.Rectify{}, action)
		rect := action.(*actions.Rectify)
		// currently no easy way to tell if the deploy client is live or dummy
		return nil, rect.Resolver.Registry
	}

	_, r := testDryRun("both")
	assert.IsType(&sous.DummyRegistry{}, r)

	_, r = testDryRun("none")
	assert.IsType(&docker.NameCache{}, r)

	_, r = testDryRun("scheduler")
	assert.IsType(&docker.NameCache{}, r)

	_, r = testDryRun("registry")
	assert.IsType(&sous.DummyRegistry{}, r)
}
