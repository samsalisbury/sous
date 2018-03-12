package sous

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
