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
		Repo:    "github.com/example/project",
		Offset:  "",
		Flavor:  "vanilla",
		Tag:     "0.1.22",
		Cluster: "test-cluster",
		All:     false,
	}
}

func fixtureGraph() *SousGraph {
	graph := DefaultTestGraph()
	graph.Add(VerbosityOverride{})
	return graph
}

func TestActionUpdate(t *testing.T) {
	action := fixtureGraph().GetUpdate(fixtureDeployFilterFlags(), config.OTPLFlags{})
	update, rightType := action.(*actions.Update)
	require.True(t, rightType)

	require.NotNil(t, update)
	require.NotNil(t, update.Manifest)
	assert.Equal(t, "github.com/example/project", update.Manifest.ID().Source.Repo)

	require.NotNil(t, update.ResolveFilter)
	require.NotNil(t, update.ResolveFilter.Repo)
	assert.Equal(t, "github.com/example/project", update.ResolveFilter.Repo.ValueOr("{globbed!!!}"))

	// these need more specific tests than "NotNil"
	require.NotNil(t, update.GDM)
	require.NotNil(t, update.User)
	require.NotNil(t, update.Client)
}

func TestActionPollStatus(t *testing.T) {
	action := fixtureGraph().GetPollStatus("both", fixtureDeployFilterFlags())
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
	require.NotNil(t, rect.Resolver.Filter)
	assert.Equal(t, rect.Resolver.Filter.All(), false)
}

func TestActionRectifyDryruns(t *testing.T) {
	testDryRun := func(which string, expectedRegistryType sous.Registry) {
		t.Run("dryrun is "+which, func(t *testing.T) {
			action := fixtureGraph().GetRectify(which, fixtureDeployFilterFlags())
			require.IsType(t, &actions.Rectify{}, action)
			rect := action.(*actions.Rectify)
			require.NotNil(t, rect.Resolver)
			assert.IsType(t, expectedRegistryType, rect.Resolver.Registry)
		})
	}

	testDryRun("both", &sous.DummyRegistry{})
	testDryRun("registry", &sous.DummyRegistry{})
	testDryRun("none", &docker.NameCache{})
	testDryRun("scheduler", &docker.NameCache{})
}
