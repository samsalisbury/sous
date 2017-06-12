package sous

type (
	// StateReader knows how to read state.
	StateReader interface {
		ReadState() (*State, error)
	}
	// StateWriter knows how to write state.
	StateWriter interface {
		WriteState(*State, StateWriteContext) error
	}

	// A StateManager can read and write state
	StateManager interface {
		StateReader
		StateWriter
	}

	// StateWriteContext contains additional data about what is being written.
	StateWriteContext struct {
		// User is the user this write is attributed to.
		User User
		// TargetManifestID is the manifest this write is expected to affect.
		// Implementations of StateWriter.WriteState should check that the
		// change being written corresponds with this manifest ID.
		TargetManifestID ManifestID
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
func (sm *DummyStateManager) WriteState(s *State, c StateWriteContext) error {
	sm.WriteCount++
	*sm.State = *s
	return nil
}
