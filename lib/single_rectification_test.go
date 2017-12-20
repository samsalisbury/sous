package sous

import (
	"testing"
	"time"
)

func TestSingleRectification_Resolve_completes(t *testing.T) {

	// This test just checks that SingleRectification.Resolve actually
	// completes.

	sr := NewSingleRectification(DeployablePair{})

	done := make(chan struct{})

	go func() {
		sr.Begin(&DummyDeployer{})
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
