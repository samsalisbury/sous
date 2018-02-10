package sous

import (
	"fmt"
	"testing"
)

func TestDeploymentID_Digest(t *testing.T) {
	d := &DeploymentID{
		ManifestID: ManifestID{
			Source: SourceLocation{
				Repo: "fake.tld/org/" + "project",
				Dir:  "down/here",
			},
		},
		Cluster: "test-cluster",
	}
	got := fmt.Sprintf("%x", d.Digest())
	const want = "3ea161adca77a01781628e8a7d24ad0e"
	if got != want {
		t.Fatalf("got: %q; want: %q", got, want)
	}
	t.Logf("success: %q mapped to %q", got, want)
}

// deploymentIDTestCases returns test cases for use by both String and Parse
// tests.
func deploymentIDTestCases() []struct {
	desc         string
	deploymentID DeploymentID
	string       string
} {
	return []struct {
		desc         string
		deploymentID DeploymentID
		string       string
	}{
		{
			desc:         "zero DeploymentID",
			deploymentID: DeploymentID{},
			string:       ":",
		},
		{
			desc: "cluster",
			deploymentID: DeploymentID{
				Cluster: "cluster1",
			},
			string: "cluster1:",
		},
		{
			desc: "cluster-repo",
			deploymentID: DeploymentID{
				ManifestID: ManifestID{
					Source: SourceLocation{
						Repo: "repo1",
					},
				},
				Cluster: "cluster1",
			},
			string: "cluster1:repo1",
		},
		{
			desc: "cluster-repo-dir",
			deploymentID: DeploymentID{
				ManifestID: ManifestID{
					Source: SourceLocation{
						Repo: "repo1",
						Dir:  "dir1",
					},
				},
				Cluster: "cluster1",
			},
			string: "cluster1:repo1,dir1",
		},
		{
			desc: "cluster-repo-dir-flavor",
			deploymentID: DeploymentID{
				ManifestID: ManifestID{
					Source: SourceLocation{
						Repo: "repo1",
						Dir:  "dir1",
					},
					Flavor: "flavor1",
				},
				Cluster: "cluster1",
			},
			string: "cluster1:repo1,dir1~flavor1",
		},
	}
}

func TestDeploymentID_String(t *testing.T) {
	for _, tc := range deploymentIDTestCases() {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.deploymentID.String()
			want := tc.string
			if got != want {
				t.Errorf("got %q; want %q", got, want)
			}
		})
	}
}

func TestParseDeploymentID_success(t *testing.T) {
	for _, tc := range deploymentIDTestCases() {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := ParseDeploymentID(tc.string)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			want := tc.deploymentID
			if got != want {
				t.Errorf("got %#v; want %#v", got, want)
			}
		})
	}
}

func TestParseDeploymentID_errors(t *testing.T) {
	testCases := []struct {
		in, want string
	}{
		{
			in:   "",
			want: "empty string not valid",
		},
		{
			in:   ":~~",
			want: `illegal manifest ID "~~" (contains more than one colon)`,
		},
		{
			in:   "nocolon",
			want: "does not contain a colon",
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%q", tc.in), func(t *testing.T) {
			gotDeploymentID, gotErr := ParseDeploymentID(tc.in)
			if gotErr == nil {
				t.Fatalf("got nil error; want %q", tc.want)
			}
			got := gotErr.Error()
			if got != tc.want {
				t.Errorf("got error %q; want %q", got, tc.want)
			}
			if (gotDeploymentID != DeploymentID{}) {
				t.Errorf("got nonzero DeploymentID %#v", gotDeploymentID)
			}
		})
	}
}
