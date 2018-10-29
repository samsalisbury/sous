package docker

import (
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

func TestImageRepoName(t *testing.T) {

	cases := []struct {
		kind string
		in   sous.SourceLocation
		want string
	}{
		// Kind: docker
		{
			"docker",
			sous.SourceLocation{Repo: "github.com/user1/repo1"},
			"example.org/user1/repo1-docker",
		},
		{
			"docker",
			sous.SourceLocation{Repo: "github.com/User1/Repo1"},
			"example.org/user1/repo1-docker",
		},
		{
			"docker",
			sous.SourceLocation{Repo: "github.com/opentable/repo1"},
			"example.org/repo1-docker",
		},
		{
			"docker",
			sous.SourceLocation{
				Repo: "github.com/opentable/repo1",
				Dir:  "dir1",
			},
			"example.org/repo1/dir1-docker",
		},
		{
			"docker",
			sous.SourceLocation{
				Repo: "github.com/user1/repo1",
				Dir:  "dir1",
			},
			"example.org/user1/repo1/dir1-docker",
		},

		// With blank kind.
		{
			"",
			sous.SourceLocation{Repo: "github.com/user1/repo1"},
			"example.org/user1/repo1",
		},
		{
			"",
			sous.SourceLocation{Repo: "github.com/User1/Repo1"},
			"example.org/user1/repo1",
		},
		{
			"",
			sous.SourceLocation{Repo: "github.com/opentable/repo1"},
			"example.org/repo1",
		},
		{
			"",
			sous.SourceLocation{
				Repo: "github.com/opentable/repo1",
				Dir:  "dir1",
			},
			"example.org/repo1/dir1",
		},
		{
			"",
			sous.SourceLocation{
				Repo: "github.com/user1/repo1",
				Dir:  "dir1",
			},
			"example.org/user1/repo1/dir1",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.kind+"/"+tc.want, func(t *testing.T) {
			got := fullRepoName("example.org", tc.in, tc.kind, stripRE, logging.SilentLogSet())
			if got != tc.want {
				t.Errorf("got %s --> %q; want %q", tc.in, got, tc.want)
			}
		})
	}

}
