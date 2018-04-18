package sous

import (
	"testing"
	"time"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/util/logging"
)

func TestSingleRectification_Resolve_completes(t *testing.T) {

	// This test just checks that SingleRectification.Resolve actually
	// completes.

	log, _ := logging.NewLogSinkSpy()
	sr := NewRectification(DeployablePair{
		Post: &Deployable{
			Deployment: &Deployment{},
		},
	}, log,
	)

	done := make(chan struct{})
	dpr, c := NewDeployerSpy()
	c.MatchMethod("Status", spies.AnyArgs, &DeployState{Status: DeployStatusActive}, nil)
	c.MatchMethod("Rectify", spies.AnyArgs, DiffResolution{}, nil)

	sr.Begin(dpr, &DummyRegistry{}, &ResolveFilter{}, NewDummyStateManager())

	go func() {
		sr.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		// A second is total overkill...
		t.Errorf("resolution took more than a second")
	}

	if err := sr.Resolution.Error; err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if sr.Resolution.DeployState == nil {
		t.Fatalf("got nil DeployState")
	}

	if sr.Resolution.DeployState.Status != DeployStatusActive {
		t.Errorf("got DeployStatus %q; want %q", sr.Resolution.DeployState.Status, DeployStatusActive)
	}
}
