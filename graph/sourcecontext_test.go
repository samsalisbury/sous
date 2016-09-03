package graph

import (
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
)

type resolveSourceLocationInput struct {
	Flags   *config.DeployFilterFlags
	Context *sous.SourceContext
}

var badResolveSourceLocationCalls = map[string][]resolveSourceLocationInput{
	"no repo specified, please use -repo or run sous inside a git repo": {
		{Flags: &config.DeployFilterFlags{}, Context: &sous.SourceContext{}},
		{Flags: nil, Context: &sous.SourceContext{}},
		{Flags: &config.DeployFilterFlags{}, Context: nil},
	},
	"you specified -offset but not -repo": {
		{Flags: &config.DeployFilterFlags{Offset: "some/offset"}},
	},
}

func TestResolveSourceLocation_failure(t *testing.T) {
	for expected, inGroup := range badResolveSourceLocationCalls {
		for _, in := range inGroup {
			_, actualErr := newTargetSourceLocation(in.Flags, in.Context)
			if actualErr == nil {
				t.Errorf("got nil; want error %q", expected)
				continue
			}
			actual := actualErr.Error()
			if actual != expected {
				t.Errorf("got error %q; want error %q", actual, expected)
			}
		}
	}
}

var goodResolveSourceLocationCalls = map[sous.SourceLocation][]resolveSourceLocationInput{
	{Repo: "github.com/user/project", Dir: ""}: {
		{Flags: &config.DeployFilterFlags{Repo: "github.com/user/project"}},
		{Context: &sous.SourceContext{PrimaryRemoteURL: "github.com/user/project"}},
	},
	{Repo: "github.com/user/project", Dir: "some/path"}: {
		{Flags: &config.DeployFilterFlags{Repo: "github.com/user/project", Offset: "some/path"}},
		{Context: &sous.SourceContext{
			PrimaryRemoteURL: "github.com/user/project",
			OffsetDir:        "some/path",
		}},
	},
	{Repo: "github.com/from/flags", Dir: ""}: {
		{
			Context: &sous.SourceContext{
				PrimaryRemoteURL: "github.com/original/context",
				OffsetDir:        "the/detected/offset",
			},
			Flags: &config.DeployFilterFlags{
				Repo: "github.com/from/flags",
			},
		},
	},
}

func TestResolveSourceLocation_success(t *testing.T) {
	for expected, inGroup := range goodResolveSourceLocationCalls {
		for _, in := range inGroup {
			actual, err := newTargetSourceLocation(in.Flags, in.Context)
			if err != nil {
				t.Error(err)
				continue
			}
			if actual.Repo != expected.Repo {
				t.Errorf("got repo %q; want %q", actual.Repo, expected.Repo)
			}
			if actual.Dir != expected.Dir {
				t.Errorf("got offset %q; want %q", actual.Dir, expected.Dir)
			}
		}
	}
}
