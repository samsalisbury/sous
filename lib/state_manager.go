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
	// User represents a user of the Sous client.
	User struct {
		// Name is the full name of this user.
		Name,
		// Email is the email address of this user.
		Email string
	}
)

// ReadState implements StateManager
func (sm *DummyStateManager) ReadState() (*State, error) {
	sm.ReadCount++
	return sm.State, nil
}

// WriteState implements StateManager
func (sm *DummyStateManager) WriteState(s *State) error {
	sm.WriteCount++
	*sm.State = *s
	return nil
}
