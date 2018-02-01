package sous

import (
	"fmt"
	"testing"
)

func TestDeploymentID_Digest(t *testing.T) {
	tmpl := "got:%s expected:%s"
	expected := "3ea161adca77a01781628e8a7d24ad0e"
	d := &DeploymentID{
		ManifestID: ManifestID{
			Source: SourceLocation{
				Repo: "fake.tld/org/" + "project",
				Dir:  "down/here",
			},
		},
		Cluster: "test-cluster",
	}
	dStr := fmt.Sprintf("%x", d.Digest())
	if dStr != expected {
		t.Fatalf(tmpl, dStr, expected)
	} else {
		t.Logf(tmpl, dStr, expected)
	}
}
