package queries

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestDeployment_Result(t *testing.T) {
	sm := sous.NewDummyStateManager()
	sm.State = sous.DefaultStateFixture()
	aq := ArtifactQuery{}
	q := Deployment{
		StateManager:  sm,
		ArtifactQuery: aq,
	}
	r, err := q.Result(DeploymentFilters{})
	if err != nil {
		t.Fatal(err)
	}
	want := 9 // NOTE SS: sous.DefaultStateFixture returns 9 deployments as standard.
	got := r.Deployments.Len()
	if got != want {
		t.Errorf("got %d deployments; want %d", got, want)
	}
}
