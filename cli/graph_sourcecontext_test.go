package cli

import (
	"testing"

	"github.com/opentable/sous/lib"
)

type resolveSourceLocationInput struct {
	Flags   *DeployFilterFlags
	Context *sous.SourceContext
}

var badResolveSourceLocationCalls = map[string][]resolveSourceLocationInput{
	"no repo specified, please use -repo or run sous inside a git repo": {
		{Flags: &DeployFilterFlags{}, Context: &sous.SourceContext{}},
		{Flags: nil, Context: &sous.SourceContext{}},
		{Flags: &DeployFilterFlags{}, Context: nil},
	},
	"you specified -offset but not -repo": {
		{Flags: &DeployFilterFlags{Offset: "some/offset"}},
	},
}

func TestResolveSourceLocation_failure(t *testing.T) {
	for expected, inGroup := range badResolveSourceLocationCalls {
		for _, in := range inGroup {
			_, actualErr := resolveSourceLocation(in.Flags, in.Context)
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
	{RepoURL: "github.com/user/project", RepoOffset: ""}: {
		{Flags: &DeployFilterFlags{Repo: "github.com/user/project"}},
		{Context: &sous.SourceContext{PrimaryRemoteURL: "github.com/user/project"}},
	},
	{RepoURL: "github.com/user/project", RepoOffset: "some/path"}: {
		{Flags: &DeployFilterFlags{Repo: "github.com/user/project", Offset: "some/path"}},
		{Context: &sous.SourceContext{
			PrimaryRemoteURL: "github.com/user/project",
			OffsetDir:        "some/path",
		}},
	},
	{RepoURL: "github.com/from/flags", RepoOffset: ""}: {
		{
			Context: &sous.SourceContext{
				PrimaryRemoteURL: "github.com/original/context",
				OffsetDir:        "the/detected/offset",
			},
			Flags: &DeployFilterFlags{
				Repo: "github.com/from/flags",
			},
		},
	},
}

func TestResolveSourceLocation_success(t *testing.T) {
	for expected, inGroup := range goodResolveSourceLocationCalls {
		for _, in := range inGroup {
			actual, err := resolveSourceLocation(in.Flags, in.Context)
			if err != nil {
				t.Error(err)
				continue
			}
			if actual.RepoURL != expected.RepoURL {
				t.Errorf("got repo %q; want %q", actual.RepoURL, expected.RepoURL)
			}
			if actual.RepoOffset != expected.RepoOffset {
				t.Errorf("got offset %q; want %q", actual.RepoOffset, expected.RepoOffset)
			}
		}
	}
}
