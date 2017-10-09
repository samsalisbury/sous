package graph

import (
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resolveSourceLocationInput struct {
	Flags   *sous.ResolveFilter
	Context *sous.SourceContext
}

func TestResolveSourceLocation_failure(t *testing.T) {
	assertSourceContextError := func(t *testing.T, flags *sous.ResolveFilter, ctx *SourceContextDiscovery, msgPattern string) {
		_, actualErr := newRefinedResolveFilter(flags, ctx)
		require.NotNil(t, actualErr)
		assert.Regexp(t, msgPattern, actualErr.Error())
	}

	assertSourceContextError(t, &sous.ResolveFilter{}, &SourceContextDiscovery{},
		"no repo specified, please use -repo or run sous inside a git repo with a configured remote")
	assertSourceContextError(t, nil, &SourceContextDiscovery{},
		"no repo specified, please use -repo or run sous inside a git repo with a configured remote")
}

func assertSourceContextSuccess(t *testing.T, expected sous.ManifestID, flags *sous.ResolveFilter, ctx *sous.SourceContext) {
	disco := &SourceContextDiscovery{SourceContext: ctx}
	rrf, err := newRefinedResolveFilter(flags, disco)
	require.NoError(t, err)

	actual, err := newTargetManifestID(rrf)
	assert.NoError(t, err)
	assert.Equal(t, actual.Source.Repo, expected.Source.Repo, "repos differ")
	assert.Equal(t, actual.Source.Dir, expected.Source.Dir, "offsets differ")
	assert.Equal(t, actual.Flavor, expected.Flavor, "flavors differ")
}

func TestResolveSourceLocation_success(t *testing.T) {

	// -repo set and matches detected repo, so that repo used.
	assertSourceContextSuccess(t,
		// expected
		sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "github.com/user/project",
			},
		},
		// flags
		&sous.ResolveFilter{
			Repo: sous.NewResolveFieldMatcher("github.com/user/project"),
		},
		// context
		&sous.SourceContext{
			PrimaryRemoteURL: "github.com/user/project",
		},
	)

	// -repo and -offset set, providing full SourceLocation.
	assertSourceContextSuccess(t,
		// expected
		sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "github.com/user/project",
				Dir:  "some/path",
			},
		},
		// flags
		&sous.ResolveFilter{
			Repo:   sous.NewResolveFieldMatcher("github.com/user/project"),
			Offset: sous.NewResolveFieldMatcher("some/path"),
		},
		// context
		&sous.SourceContext{
			PrimaryRemoteURL: "github.com/user/project",
			OffsetDir:        "some/path",
		},
	)

	// -repo set explicitly, therefore detected offset ignored.
	assertSourceContextSuccess(t,
		// expected
		sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "github.com/from/flags",
			},
		},
		// flags
		&sous.ResolveFilter{
			Repo: sous.NewResolveFieldMatcher("github.com/from/flags"),
		},
		// context
		&sous.SourceContext{
			PrimaryRemoteURL: "github.com/original/context",
			OffsetDir:        "the/detected/offset",
		},
	)
}
