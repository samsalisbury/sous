package sous

import (
	"os"
	"testing"
	"time"

	"github.com/nyarly/testify/assert"
)

func dummyResolver() *Resolver {
	return NewResolver(NewDummyDeployer(), NewDummyRegistry())
}

func setupAR() *AutoResolver {
	ls := SilentLogSet()
	return NewAutoResolver(dummyResolver(), &DummyStateManager{State: NewState()}, &ls)
}

func TestDone(t *testing.T) {
	assert := assert.New(t)

	ar := setupAR()

	received := false
	var result error

	ar.addListener(func(tc, done triggerChannel, ec announceChannel) {
		select {
		case err := <-ec:
			received = true
			result = err
			close(done)
		}
	})
	done := ar.Kickoff()
	for range done {
	}
	assert.True(received, "Should have received announcement")
}

func TestAfterDone(t *testing.T) {
	ar := setupAR()
	ar.UpdateTime = time.Duration(1)

	tc := make(triggerChannel, 1)
	ac := make(announceChannel, 1)
	done := make(triggerChannel, 1)

	ac <- nil
	ar.afterDone(tc, done, ac)
	select {
	case <-tc:
	case <-time.After(time.Duration(2)):
		t.Error("Trigger channel took too long")
	default:
		t.Error("Trigger channel not triggered")
	}
}

func TestResolveLoop(t *testing.T) {
	ar := setupAR()
	ar.LogSet.Debug.SetOutput(os.Stderr)

	tc := make(triggerChannel, 10)
	ac := make(announceChannel, 1)
	done := make(triggerChannel)

	for i := 0; i < 10; i++ {
		tc.trigger()
	}
	ar.resolveLoop(tc, done, ac)

	select {
	case <-tc:
		t.Error("Should have consumed all the triggers")
	default:
	}

	select {
	case <-ac:
	default:
		t.Error("Should have announced a result")
	}
}
