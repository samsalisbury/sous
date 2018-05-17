package sous

import "github.com/nyarly/spies"

type (
	// StateReader knows how to read state.
	StateReader interface {
		ReadState() (*State, error)
	}
	// StateWriter knows how to write state.
	StateWriter interface {
		WriteState(*State, User) error
	}

	// A StateManager can read and write state
	StateManager interface {
		StateReader
		StateWriter
	}

	// DummyStateManager is used for testing
	DummyStateManager struct {
		*State
		ReadCount, WriteCount int
		WriteErr              error
		ReadErr               error
	}

	// StateManagerSpy is a spy implementation of StateManager.
	StateManagerSpy struct {
		spy *spies.Spy
	}

	// StateManagerController is the controller for a spy implementation of StateManager.
	StateManagerController struct {
		*spies.Spy
	}
)

// NewDummyStateManager returns a dummy StateManager, suitable for testing.
func NewDummyStateManager() *DummyStateManager {
	return &DummyStateManager{State: NewState()}
}

// ReadState implements StateManager
func (sm *DummyStateManager) ReadState() (*State, error) {
	sm.ReadCount++
	return sm.State, sm.ReadErr
}

// WriteState implements StateManager
func (sm *DummyStateManager) WriteState(s *State, u User) error {
	sm.WriteCount++
	*sm.State = *s
	return sm.WriteErr
}

// NewStateManagerSpy creates a StateManager spy.
func NewStateManagerSpy() (StateManager, StateManagerController) {
	spy := spies.NewSpy()

	return StateManagerSpy{spy: spy}, StateManagerController{Spy: spy}
}

// NewStateManagerSpyFor returns a spy/controller pair that will return the given state for calls to Read
func NewStateManagerSpyFor(state *State) (StateManager, StateManagerController) {
	sm, sc := NewStateManagerSpy()
	sc.MatchMethod("Read", spies.AnyArgs, state, nil)
	return sm, sc
}

// ReadState implements StateManager on StateManagerSpy.
func (spy StateManagerSpy) ReadState() (*State, error) {
	res := spy.spy.Called()
	return res.Get(0).(*State), res.Error(1)
}

// WriteState implements StateManager on StateManagerSpy.
func (spy StateManagerSpy) WriteState(s *State, u User) error {
	res := spy.spy.Called(s, u)
	return res.Error(0)
}
