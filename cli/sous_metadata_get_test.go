package cli

import (
	"bytes"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful/restfultest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataGetFieldAndCluster(t *testing.T) {
	output := runCommand(t, []string{"DeployOn"}, config.DeployFilterFlags{
		Repo:    project1.Repo,
		Cluster: "cluster-2",
	})

	assert.Regexp(t, "version advance", output)
}

func TestMetadataGetAllCluster(t *testing.T) {
	output := runCommand(t, []string{}, config.DeployFilterFlags{
		Repo:    project1.Repo,
		Cluster: "cluster-2",
	})

	assert.Regexp(t, "BuildBranch", output)
	assert.Regexp(t, "master", output)
	assert.Regexp(t, "DeployOn", output)
	assert.Regexp(t, "version advance", output)
	assert.NotRegexp(t, "build success", output)
}

func TestMetadataGetField(t *testing.T) {
	output := runCommand(t, []string{"BuildBranch"}, config.DeployFilterFlags{
		Repo: project1.Repo,
	})

	assert.Regexp(t, "master", output)
}

func TestMetadataGetAll(t *testing.T) {
	output := runCommand(t, []string{}, config.DeployFilterFlags{
		Repo: project1.Repo,
	})

	assert.Regexp(t, "BuildBranch", output)
	assert.Regexp(t, "master", output)
	assert.Regexp(t, "DeployOn", output)
	assert.Regexp(t, "version advance", output)
	assert.Regexp(t, "build success", output)
}

func runCommand(t *testing.T, args []string, dff config.DeployFilterFlags) string {
	cl, control := restfultest.NewHTTPClientSpy()
	out := &bytes.Buffer{}
	shc := sous.SourceHostChooser{}
	rf, err := dff.BuildFilter(shc.ParseSourceLocation)
	require.NoError(t, err)
	smg := SousMetadataGet{
		TargetManifestID:  graph.TargetManifestID{Source: sous.SourceLocation{Repo: project1.Repo}},
		DeployFilterFlags: dff,
		ResolveFilter:     rf,
		HTTPClient:        graph.HTTPClient{cl},
		OutWriter:         graph.OutWriter(out),
	}

	control.Any(
		"Retrieve",
		sous.ManifestFixture("with-metadata"), restfultest.DummyUpdater(), nil,
	)
	res := smg.Execute(args)
	assert.Equal(t, 0, res.ExitCode())

	return out.String()
}

var project1 = sous.SourceLocation{Repo: "github.com/user/project"}

func makeTestState() *sous.State {
	cluster1 := &sous.Cluster{
		Name:    "cluster-1",
		Kind:    "singularity",
		BaseURL: "http://nothing.here.one",
		Env: sous.EnvDefaults{
			"CLUSTER_LONG_NAME": sous.Var("Cluster One"),
		},
	}
	cluster2 := &sous.Cluster{
		Name:    "cluster-2",
		Kind:    "singularity",
		BaseURL: "http://nothing.here.two",
		Env: sous.EnvDefaults{
			"CLUSTER_LONG_NAME": sous.Var("Cluster Two"),
		},
	}
	return &sous.State{
		Defs: sous.Defs{
			DockerRepo: "some.docker.repo",
			Clusters: sous.Clusters{
				"cluster-1": cluster1,
				"cluster-2": cluster2,
			},
		},
		Manifests: sous.NewManifests(sous.ManifestFixture("with-metadata")),
	}
}
