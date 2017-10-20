package sous

import (
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func dummyResolver() *Resolver {
	return NewResolver(NewDummyDeployer(), NewDummyRegistry(), &ResolveFilter{}, logging.SilentLogSet())
}

func setupAR() *AutoResolver {
	ls := logging.SilentLogSet()
	return NewAutoResolver(dummyResolver(), &DummyStateManager{State: NewState()}, ls)
}

func TestDone(t *testing.T) {
	assert := assert.New(t)

	ar := setupAR()

	received := false

	ar.addListener(func(tc, done TriggerChannel, ec announceChannel) {
		select {
		case <-ec:
			received = true
			close(done)
		}
	})
	done := ar.Kickoff()
	for range done {
	}
	assert.True(received, "Should have received announcement")
}

func TestAutoResolver_CurrentStatus(t *testing.T) {
	assert := assert.New(t)
	ar := setupAR()

	tc := make(TriggerChannel, 10)
	ac := make(announceChannel, 1)
	done := make(TriggerChannel)
	tc.trigger()

	stable, live := ar.Statuses()
	assert.Nil(stable)
	assert.Nil(live)

	ar.resolveLoop(tc, done, ac)
	stable, live = ar.Statuses()
	assert.NotNil(stable)
	assert.NotNil(live)
}

func TestAfterDone(t *testing.T) {
	ar := setupAR()
	ar.UpdateTime = time.Duration(1)

	tc := make(TriggerChannel, 1)
	ac := make(announceChannel, 1)
	done := make(TriggerChannel, 1)

	ac <- nil
	ar.afterDone(tc, done, ac)
	select {
	case <-tc:
	case <-time.After(time.Duration(8 * time.Millisecond)):
		t.Error("Trigger channel took too long")
	default:
		t.Error("Trigger channel not triggered")
	}
}

func TestResolveLoop(t *testing.T) {
	ar := setupAR()

	tc := make(TriggerChannel, 10)
	ac := make(announceChannel, 1)
	done := make(TriggerChannel)

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
