package docker

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/distribution/reference"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

func TestFullRepoName_github(t *testing.T) {

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
			got := fullRepoName("example.org", tc.in, tc.kind, logging.SilentLogSet())
			if got != tc.want {
				t.Errorf("got %s --> %q; want %q", tc.in, got, tc.want)
			}
		})
	}

}

func TestImageRepoName_github(t *testing.T) {

	cases := []struct {
		kind string
		in   sous.SourceLocation
		want string
	}{
		// Kind: docker
		{
			"docker",
			sous.SourceLocation{Repo: "github.com/user1/repo1"},
			"/user1/repo1-docker",
		},
		{
			"docker",
			sous.SourceLocation{Repo: "github.com/User1/Repo1"},
			"/user1/repo1-docker",
		},
		{
			"docker",
			sous.SourceLocation{Repo: "github.com/opentable/repo1"},
			"repo1-docker",
		},
		{
			"docker",
			sous.SourceLocation{
				Repo: "github.com/opentable/repo1",
				Dir:  "dir1",
			},
			"repo1/dir1-docker",
		},
		{
			"docker",
			sous.SourceLocation{
				Repo: "github.com/user1/repo1",
				Dir:  "dir1",
			},
			"/user1/repo1/dir1-docker",
		},

		// With blank kind.
		{
			"",
			sous.SourceLocation{Repo: "github.com/user1/repo1"},
			"/user1/repo1",
		},
		{
			"",
			sous.SourceLocation{Repo: "github.com/User1/Repo1"},
			"/user1/repo1",
		},
		{
			"",
			sous.SourceLocation{Repo: "github.com/opentable/repo1"},
			"repo1",
		},
		{
			"",
			sous.SourceLocation{
				Repo: "github.com/opentable/repo1",
				Dir:  "dir1",
			},
			"repo1/dir1",
		},
		{
			"",
			sous.SourceLocation{
				Repo: "github.com/user1/repo1",
				Dir:  "dir1",
			},
			"/user1/repo1/dir1",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.kind+"/"+tc.want, func(t *testing.T) {
			got := imageRepoName(tc.in, tc.kind)
			if got != tc.want {
				t.Errorf("got %s --> %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestImageRepoName_generic(t *testing.T) {

	type testCaseInput struct {
		kind string
		sl   sous.SourceLocation
	}
	cases := []testCaseInput{
		// Kind: docker
		{
			"docker",
			sous.SourceLocation{Repo: "example.org/user1/repo1"},
		},
		{
			"docker",
			sous.SourceLocation{Repo: "example.org/User1/Repo1"},
		},
		{
			"docker",
			sous.SourceLocation{Repo: "example.org/opentable/repo1"},
		},
		{
			"docker",
			sous.SourceLocation{
				Repo: "example.org/opentable/repo1",
				Dir:  "dir1",
			},
		},
		{
			"docker",
			sous.SourceLocation{
				Repo: "example.org/user1/repo1",
				Dir:  "dir1",
			},
		},

		// With blank kind.
		{
			"",
			sous.SourceLocation{Repo: "example.org/user1/repo1"}},
		{
			"",
			sous.SourceLocation{Repo: "example.org/User1/Repo1"}},
		{
			"",
			sous.SourceLocation{Repo: "example.org/opentable/repo1"}},
		{
			"",
			sous.SourceLocation{
				Repo: "example.org/opentable/repo1",
				Dir:  "dir1",
			},
		},
		{
			"",
			sous.SourceLocation{
				Repo: "example.org/user1/repo1",
				Dir:  "dir1",
			},
		},
	}

	assertKindSuffix := func(t *testing.T, out, kind string) {
		if kind == "" {
			return
		}
		wantSuffix := "-" + kind
		if !strings.HasSuffix(out, wantSuffix) {
			t.Errorf("got %q; want suffix %q", out, wantSuffix)
		}
	}

	assertValidDockerRef := func(t *testing.T, out string) {
		_, err := reference.ParseNamed(out)
		if err != nil {
			t.Errorf("invalid docker ref: %s", err)
		}
	}

	// Assert that given all inputs are unique, all outputs are also unique.
	gotInputs := map[testCaseInput]int{}
	gotOutputs := map[string]int{}
	for i, tc := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {

			// Uniqueness.
			if firstIndex, ok := gotInputs[tc]; ok {
				t.Fatalf("Test cases %d and %d have duplicate inputs.", firstIndex, i)
			}
			gotInputs[tc] = i
			out := imageRepoName(tc.sl, tc.kind)
			if firstIndex, ok := gotOutputs[out]; ok {
				t.Errorf("Test cases %d and %d produced the same output: %q", firstIndex, i, out)
			}
			gotOutputs[out] = i

			// Extra assertions.
			assertKindSuffix(t, out, tc.kind)
			assertValidDockerRef(t, out)
		})
	}
}
