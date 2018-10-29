package docker

import (
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

func TestImageRepoName(t *testing.T) {

	cases := []struct {
		in   sous.SourceLocation
		want string
	}{
		{
			sous.SourceLocation{Repo: "github.com/user1/repo1"},
			"example.org/user1/repo1-docker",
		},
		{
			sous.SourceLocation{Repo: "github.com/User1/Repo1"},
			"example.org/user1/repo1-docker",
		},
		{
			sous.SourceLocation{Repo: "github.com/opentable/repo1"},
			"example.org/repo1-docker",
		},
		{
			sous.SourceLocation{
				Repo: "github.com/opentable/repo1",
				Dir:  "dir1",
			},
			"example.org/repo1/dir1-docker",
		},
		{
			sous.SourceLocation{
				Repo: "github.com/user1/repo1",
				Dir:  "dir1",
			},
			"example.org/user1/repo1/dir1-docker",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			got := fullRepoName("example.org", tc.in, "docker", stripRE, logging.SilentLogSet())
			if got != tc.want {
				t.Errorf("got %s --> %q; want %q", tc.in, got, tc.want)
			}
		})
	}

}
