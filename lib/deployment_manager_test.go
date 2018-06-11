package sous

import (
	"testing"

	"github.com/opentable/sous/util/logging"
)

func TestDeploymentManager_ReadDeployment(t *testing.T) {

	innerState := DefaultStateFixture()
	dummy := &DummyStateManager{
		State: innerState,
	}
	ls, _ := logging.NewLogSinkSpy()
	dm := MakeDeploymentManager(dummy, ls)

	did := DeploymentID{
		ManifestID: ManifestID{
			Source: SourceLocation{
				Repo: "github.com/user1/repo1",
				Dir:  "dir1",
			},
			Flavor: "flavor1",
		},
		Cluster: "cluster1",
	}
	originalDeployments, err := innerState.Deployments()
	if err != nil {
		t.Fatal(err)
	}
	originalDeployment, ok := originalDeployments.Snapshot()[did]
	if !ok {
		t.Fatalf("setup failed: no deployment matching %q", did)
	}
	deployment, err := dm.ReadDeployment(did)
	if err != nil {
		t.Fatal(err)
	}

	// XXX uses deployment.Diff
	different, diffs := deployment.Diff(originalDeployment)
	if different {
		t.Errorf("ReadDeployment returned different deployment (diffs: %#v)", diffs)
	}
}
