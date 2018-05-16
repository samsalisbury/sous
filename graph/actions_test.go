package graph

import (
	"testing"

	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/docker"
	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/psyringe"
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

func fixtureGraph(t *testing.T) *SousGraph {
	graph := DefaultTestGraph(t)
	graph.Add(&config.Verbosity{})
	return graph
}

func TestActionUpdate(t *testing.T) {
	fg := fixtureGraph(t)
	flags := fixtureDeployFilterFlags()
	action, err := fg.GetUpdate(flags, config.OTPLFlags{})
	require.NoError(t, err)
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
	require.NotNil(t, update.HTTPStateManager)
}

func TestActionPollStatus(t *testing.T) {
	fg := fixtureGraph(t)
	fg.Add(fixtureDeployFilterFlags())

	action, err := fg.GetPollStatus("both", fixtureDeployFilterFlags())
	require.NoError(t, err)
	pollStatus, rightType := action.(*actions.PollStatus)
	require.True(t, rightType)

	require.NotNil(t, pollStatus.StatusPoller)
	assert.NotEqual(t, "", pollStatus.StatusPoller.Repo)
	assert.Equal(t, pollStatus.StatusPoller.ResolveFilter.Repo.ValueOr("{globbed!!!}"), fixtureDeployFilterFlags().Repo)
}

func TestActionRectify(t *testing.T) {
	fg := fixtureGraph(t)
	fg.Add(fixtureDeployFilterFlags())
	action, err := fg.GetRectify("none", fixtureDeployFilterFlags())
	require.NoError(t, err)

	rect, rightType := action.(*actions.Rectify)
	require.True(t, rightType)

	assert.NotNil(t, rect.State)
	require.NotNil(t, rect.Resolver)
	require.NotNil(t, rect.Resolver.ResolveFilter)
	assert.Equal(t, rect.Resolver.ResolveFilter.All(), false)
}

func TestActionRectifyDryruns(t *testing.T) {
	testDryRun := func(which, sousServerURL string, expectedRegistryType sous.Registry) {
		t.Run("dryrun is "+which, func(t *testing.T) {
			fg := fixtureGraph(t)
			fg.Add(fixtureDeployFilterFlags())
			tg := psyringe.TestPsyringe{Psyringe: fg.Psyringe}
			tg.Replace(func() StateReader {
				return StateReader{StateReader: sous.NewDummyStateManager()}
			})
			tg.Replace(LocalSousConfig{
				Config: &config.Config{Server: sousServerURL},
			})
			tg.Replace(func() lazyNameCache {
				return func() (*docker.NameCache, error) {
					return &docker.NameCache{}, nil
				}
			})
			action, err := fg.GetRectify(which, fixtureDeployFilterFlags())
			require.NoError(t, err)
			require.IsType(t, &actions.Rectify{}, action)
			rect := action.(*actions.Rectify)
			require.NotNil(t, rect.Resolver)
			assert.IsType(t, expectedRegistryType, rect.Resolver.Registry)
		})
	}

	testDryRun("both", "", &sous.DummyRegistry{})
	testDryRun("registry", "", &sous.DummyRegistry{})
	testDryRun("none", "", &docker.NameCache{})
	testDryRun("scheduler", "", &docker.NameCache{})
	// TODO SS: Figure out why following 4 test cases fail.
	//testDryRun("both", "not empty", &sous.DummyRegistry{})
	//testDryRun("registry", "not empty", &sous.DummyRegistry{})
	//testDryRun("none", "not empty", &sous.DummyRegistry{})
	//testDryRun("scheduler", "not empty", &sous.DummyRegistry{})
}
