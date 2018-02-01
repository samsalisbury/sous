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

func TestDeploymentID_String(t *testing.T) {
	testCases := []struct {
		desc string
		in   DeploymentID
		want string
	}{
		{
			desc: "zero DeploymentID",
			in:   DeploymentID{},
			want: ":",
		},
		{
			desc: "cluster",
			in: DeploymentID{
				Cluster: "cluster1",
			},
			want: "cluster1:",
		},
		{
			desc: "cluster-repo",
			in: DeploymentID{
				ManifestID: ManifestID{
					Source: SourceLocation{
						Repo: "repo1",
					},
				},
				Cluster: "cluster1",
			},
			want: "cluster1:repo1",
		},
		{
			desc: "cluster-repo-dir",
			in: DeploymentID{
				ManifestID: ManifestID{
					Source: SourceLocation{
						Repo: "repo1",
						Dir:  "dir1",
					},
				},
				Cluster: "cluster1",
			},
			want: "cluster1:repo1,dir1",
		},
		{
			desc: "cluster-repo-dir-flavor",
			in: DeploymentID{
				ManifestID: ManifestID{
					Source: SourceLocation{
						Repo: "repo1",
						Dir:  "dir1",
					},
					Flavor: "flavor1",
				},
				Cluster: "cluster1",
			},
			want: "cluster1:repo1,dir1~flavor1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.in.String()
			if got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}
