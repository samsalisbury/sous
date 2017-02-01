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
	}
)

// ReadState implements StateManager
func (sm *DummyStateManager) ReadState() (*State, error) {
	sm.ReadCount++
	return sm.State, nil
}

// WriteState implements StateManager
func (sm *DummyStateManager) WriteState(s *State, u User) error {
	sm.WriteCount++
	*sm.State = *s
	return nil
}
