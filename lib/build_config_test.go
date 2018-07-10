package sous

import (
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

// Things that we can't easily do yet:
// ... and we're in a git workspace ...
//    If we aren't git.NewRepo() will fail
// If it's absent, we'll be building from a shallow clone
//    Again, by the time we get here, we're in a repo already
//    So, shallow clones become tricky

// If --repo is present, and we're in a git workspace, compare the --repo to
// the remotes of the workspace. If it's present, assume that we're working in
// the current workspace.

// We're now either working locally
// (in the git workspace) or in a clone.
// If --force-clone is present, we ignore
// the presence of a valid workspace and do a shallow clone anyway.

func TestPresentExplicitRepo(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Repo: "github.com/opentable/present",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				RemoteURLs: []string{
					"github.com/opentable/present",
					"github.com/opentable/also",
				},
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Equal(`github.com/opentable/present`, ctx.Source.RemoteURL)
}

// If it's absent, we'll be building from a shallow
// clone of the given --repo.
func TestMissingExplicitRepo(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Repo: "github.com/opentable/present",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				PrimaryRemoteURL: "github.com/guessed/upstream",
				RemoteURLs: []string{
					"github.com/opentable/also",
				},
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Equal(`github.com/opentable/present`, ctx.Source.RemoteURL)
	assert.Contains(ctx.Advisories, UnknownRepo)
}

// If --repo is absent, guess the repo from the
// remotes of the current workspace: first the upstream workspace, then the
// origin.
func TestAbsentRepoConfig(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Repo: "",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				PrimaryRemoteURL: "github.com/guessed/upstream",
				RemoteURLs: []string{
					"github.com/opentable/also",
				},
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Equal(`github.com/guessed/upstream`, ctx.Source.RemoteURL)
}

// If neither are present on the current workspace (or we're not in a
// git workspace), add the advisory "no repo."
func TestNoRepo(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Repo: "",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				PrimaryRemoteURL: "",
				RemoteURLs: []string{
					"github.com/opentable/also",
				},
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Contains(ctx.Advisories, NoRepoAdv)
}

// If a revision is specified, but that's not what's checked out,
// add an advisory
func TestNotRequestedRevision(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Revision: "abcdef",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				Revision: "100100100",
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Equal(`100100100`, ctx.Source.Revision)
	assert.Contains(ctx.Advisories, NotRequestedRevision)

}

func TestUsesRequestedTag(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Tag: "1.2.3",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				Revision: "abcd",
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	// 991
	assert.Equal(`1.2.3`, ctx.Source.Version().Version.String())
}

func TestAdvisesOfDefaultVersion(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				Revision: "abcd",
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	// 991
	assert.Equal(`0.0.0-unversioned`, ctx.Source.Version().Version.String())
	assert.Contains(ctx.Advisories, Unversioned)
}

func TestTagNotHead(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Tag: "1.2.3",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				Revision:           "abcd",
				NearestTagName:     "1.2.3",
				NearestTagRevision: "def0",
				Tags: []Tag{
					Tag{Name: "1.2.3"},
				},
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	// 991
	assert.Equal(`1.2.3`, ctx.Source.Version().Version.String())
	assert.Contains(ctx.Advisories, TagNotHead)
	assert.NotContains(ctx.Advisories, EphemeralTag)
}

func TestEphemeralTag(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Tag: "1.2.3",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				PrimaryRemoteURL:   "github.com/opentable/present",
				Revision:           "abcd",
				NearestTagName:     "1.2.0",
				NearestTagRevision: "3541",
			},
		},
		LogSink: logging.SilentLogSet(),
	}
	/*
		bc := BuildConfig{
			Strict:   true,
			Tag:      "1.2.3",
			Repo:     "github.com/opentable/present",
			Revision: "abcdef",
			Context: &BuildContext{
			Sh: &shell.Sh{},
				Source: SourceContext{
					RemoteURL: "github.com/opentable/present",
					RemoteURLs: []string{
						"github.com/opentable/present",
						"github.com/opentable/also",
					},
					Revision:           "abcdef",
					NearestTagName:     "1.2.3",
					NearestTagRevision: "abcdef",
					Tags: []Tag{
						Tag{Name: "1.2.3"},
					},
				},
			},
		}
	*/

	ctx := bc.NewContext()
	br := contextualizedResults(ctx)
	// 991
	assert.Equal(`1.2.3`, ctx.Source.Version().Version.String())
	assert.Contains(ctx.Advisories, EphemeralTag)
	assert.NotContains(ctx.Advisories, TagNotHead)
	assert.NoError(bc.GuardRegister(br))
}

func TestContextualization(t *testing.T) {
	repo := "github.com/opentable/present"
	bc := BuildConfig{
		Tag: "1.2.3",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				PrimaryRemoteURL:   repo,
				Revision:           "abcd",
				NearestTagName:     "1.2.0",
				NearestTagRevision: "3541",
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()

	otherrepo := "github.com/example/elsewhere"
	otherdir := "deep/inside"
	br := &BuildResult{Products: []*BuildProduct{{}, {
		Source: SourceID{
			Location: SourceLocation{
				Repo: otherrepo,
				Dir:  otherdir,
			},
		},
	}}}

	br.Contextualize(ctx)
	assert.Len(t, br.Products, 2)
	assert.Equal(t, br.Products[0].Source.Location.Repo, repo)
	assert.Equal(t, br.Products[0].Source.Location.Dir, "")

	assert.Equal(t, br.Products[1].Source.Location.Repo, otherrepo)
	assert.Equal(t, br.Products[1].Source.Location.Dir, otherdir)
}

func contextualizedResults(ctx *BuildContext) *BuildResult {
	br := &BuildResult{
		Products: []*BuildProduct{{}},
	}
	br.Contextualize(ctx)
	return br

}

func TestSetsOffset(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Offset: "sub/",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				OffsetDir:          "",
				Revision:           "abcd",
				NearestTagName:     "1.2.0",
				NearestTagRevision: "def0",
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Equal(`sub`, ctx.Source.OffsetDir)
}

func TestDirtyWorkspaceAdvisory(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				DirtyWorkingTree: true,
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Contains(ctx.Advisories, DirtyWS)
	assert.Error(bc.GuardRegister(contextualizedResults(ctx)))
}

func TestUnpushedRevisionAdvisory(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Strict: true,
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				RevisionUnpushed: true,
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Contains(ctx.Advisories, UnpushedRev)
	assert.Error(bc.GuardStrict(ctx))
}

func TestPermissiveGuard(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Strict: false,
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				RevisionUnpushed: true,
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Contains(ctx.Advisories, UnpushedRev)
	assert.NoError(bc.GuardStrict(ctx))
}

func TestProductionReady(t *testing.T) {
	assert := assert.New(t)

	bc := BuildConfig{
		Strict:   true,
		Tag:      "1.2.3",
		Repo:     "github.com/opentable/present",
		Revision: "abcdef",
		Context: &BuildContext{
			Sh: &shell.Sh{},
			Source: SourceContext{
				RemoteURL: "github.com/opentable/present",
				RemoteURLs: []string{
					"github.com/opentable/present",
					"github.com/opentable/also",
				},
				Revision:           "abcdef",
				NearestTagName:     "1.2.3",
				NearestTagRevision: "abcdef",
				Tags: []Tag{
					Tag{Name: "1.2.3"},
				},
			},
		},
		LogSink: logging.SilentLogSet(),
	}

	ctx := bc.NewContext()
	assert.Len(ctx.Advisories, 0)
	assert.NoError(bc.GuardStrict(ctx))
}

func TestBuildConfig_GuardRegister(t *testing.T) {
	c := &BuildConfig{
		LogSink: logging.SilentLogSet(),
	}
	bc := &BuildContext{}
	bc.Advisories = Advisories{"dirty workspace"}
	err := c.GuardRegister(contextualizedResults(bc))
	expected := "build may not be deployable in all clusters due to advisories:\n  ,0.0.0-unversioned: dirty workspace"
	if err == nil {
		t.Fatalf("got nil; want error %q", expected)
	}
	actual := err.Error()
	if actual != expected {
		t.Errorf("got error %q; want %q", actual, expected)
	}
}
