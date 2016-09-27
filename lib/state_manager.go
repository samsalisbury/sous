package sous

type (
	// StateReader knows how to read state.
	StateReader interface {
		ReadState() (*State, error)
	}
	// StateWriter knows how to write state.
	StateWriter interface {
		WriteState(*State) error
	}

	// A StateManager can read and write state
	StateManager interface {
		StateReader
		StateWriter
	}

	// DummyStateManager is used for testing
	DummyStateManager struct {
		*State
	}
)

// ReadState implements StateManager
func (sm DummyStateManager) ReadState() (*State, error) {
	return sm.State, nil
}

// WriteState implements StateManager
func (sm DummyStateManager) WriteState(s *State) error {
	*sm.State = *s
	return nil
}
