package singularity

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

var requestIDTests = []struct {
	DeployID sous.DeployID
	String   string
}{
	// repo, cluster
	{
		DeployID: sous.DeployID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user/repo",
				},
			},
			Cluster: "some-cluster",
		},
		String: "github.com>user>repo::some-cluster",
	},
	// repo, dir, cluster
	{
		DeployID: sous.DeployID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user/repo",
					Dir:  "some/offset/dir",
				},
			},
			Cluster: "some-cluster",
		},
		String: "github.com>user>repo,some>offset>dir::some-cluster",
	},
	// repo, flavor, cluster
	{
		DeployID: sous.DeployID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user/repo",
				},
				Flavor: "tasty-flavor",
			},
			Cluster: "some-cluster",
		},
		String: "github.com>user>repo:tasty-flavor:some-cluster",
	},
	// repo, dir, flavor, cluster
	{
		DeployID: sous.DeployID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user/repo",
					Dir:  "some/offset/dir",
				},
				Flavor: "tasty-flavor",
			},
			Cluster: "some-cluster",
		},
		String: "github.com>user>repo,some>offset>dir:tasty-flavor:some-cluster",
	},
}

func TestMakeRequestID(t *testing.T) {
	for _, test := range requestIDTests {
		input := test.DeployID
		expected := test.String
		actual := MakeRequestID(input)
		if actual != expected {
			t.Errorf("%#v got %q; want %q", input, actual, expected)
		}
	}
}
