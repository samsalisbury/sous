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
