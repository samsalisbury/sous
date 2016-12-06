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

func TestParseRequestID(t *testing.T) {
	for _, test := range requestIDTests {
		input := test.String
		expected := test.DeployID
		actual, err := ParseRequestID(input)
		if err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("%#v got %q; want %q", input, actual, expected)
		}
	}
}

var parseRequestIDErrors = []struct {
	String string
	Error  string
}{
	{
		String: "Hi",
		Error:  `request ID "Hi" should contain exactly 2 colons`,
	},
	{
		String: ":yo:hoho",
		Error:  `request ID ":yo:hoho" has an empty SourceLocation`,
	},
	{
		String: "yo:hoho:",
		Error:  `request ID "yo:hoho:" has an empty Cluster name`,
	},
}

func TestParseRequestID_errors(t *testing.T) {
	for _, test := range parseRequestIDErrors {
		input := test.String
		expected := test.Error
		did, err := ParseRequestID(input)
		if (did != sous.DeployID{}) {
			t.Errorf("got %#v; want %#v", did, sous.DeployID{})
		}
		if err == nil {
			t.Errorf("got nil; want error %q", expected)
			continue
		}
		actual := err.Error()
		if actual != expected {
			t.Errorf("got error %q; want %q", actual, expected)
		}
	}
}

func TestRectifyRecover(t *testing.T) {
	var err error
	expected := "Panicked"
	func() {
		defer rectifyRecover("something", "TestRectifyRecover", &err)
		panic("What's that coming over the hill?!")
	}()
	if err == nil {
		t.Fatalf("got nil, want error %q", expected)
	}
	actual := err.Error()
	if actual != expected {
		t.Errorf("got error %q; want %q", actual, expected)
	}
}
