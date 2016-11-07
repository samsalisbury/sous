package graph

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/nyarly/testify/require"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
)

type resolveSourceLocationInput struct {
	Flags   *config.DeployFilterFlags
	Context *sous.SourceContext
}

func TestResolveSourceLocation_failure(t *testing.T) {
	assertSourceContextError(t, &config.DeployFilterFlags{}, &SourceContextDiscovery{}, "no repo specified, please use -repo or run sous inside a git repo")
	assertSourceContextError(t, nil, &SourceContextDiscovery{}, "no repo specified, please use -repo or run sous inside a git repo")
	assertSourceContextError(t,
		&config.DeployFilterFlags{Offset: "some/offset"},
		&SourceContextDiscovery{},
		"-offset.*without.*-repo")
}

func assertSourceContextError(t *testing.T, flags *config.DeployFilterFlags, ctx *SourceContextDiscovery, msgPattern string) {
	_, actualErr := newTargetManifestID(flags, ctx)
	assert.NotNil(t, actualErr)
	assert.Regexp(t, msgPattern, actualErr.Error())
}

func assertSourceContextSuccess(t *testing.T, expected sous.ManifestID, flags *config.DeployFilterFlags, ctx *sous.SourceContext) {
	disco := &SourceContextDiscovery{SourceContext: ctx}
	actual, err := newTargetManifestID(flags, disco)
	require.NoError(t, err)
	assert.Equal(t, actual.Source.Repo, expected.Source.Repo, "repos differ")
	assert.Equal(t, actual.Source.Dir, expected.Source.Dir, "offsets differ")
	assert.Equal(t, actual.Flavor, expected.Flavor, "flavors differ")
}

func TestResolveSourceLocation_success(t *testing.T) {
	assertSourceContextSuccess(t,
		sous.ManifestID{Source: sous.SourceLocation{Repo: "github.com/user/project"}},
		&config.DeployFilterFlags{Repo: "github.com/user/project"},
		&sous.SourceContext{PrimaryRemoteURL: "github.com/user/project"},
	)
	assertSourceContextSuccess(t,
		sous.ManifestID{Source: sous.SourceLocation{Repo: "github.com/user/project", Dir: "some/path"}},
		&config.DeployFilterFlags{Repo: "github.com/user/project", Offset: "some/path"},
		&sous.SourceContext{
			PrimaryRemoteURL: "github.com/user/project",
			OffsetDir:        "some/path",
		},
	)
	assertSourceContextSuccess(t,
		sous.ManifestID{Source: sous.SourceLocation{Repo: "github.com/from/flags"}},
		&config.DeployFilterFlags{
			Repo: "github.com/from/flags",
		},
		&sous.SourceContext{
			PrimaryRemoteURL: "github.com/original/context",
			OffsetDir:        "the/detected/offset",
		},
	)
}
