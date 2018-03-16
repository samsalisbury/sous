package sous

import (
	"testing"
	"time"

	"github.com/nyarly/spies"
)

func TestSingleRectification_Resolve_completes(t *testing.T) {

	// This test just checks that SingleRectification.Resolve actually
	// completes.

	sr := NewRectification(DeployablePair{
		Post: &Deployable{
			Deployment: &Deployment{},
		},
	})

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
}
