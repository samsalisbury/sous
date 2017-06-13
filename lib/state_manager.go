package sous

type (
	// StateReader knows how to read state.
	StateReader interface {
		ReadState(StateContext) (*State, error)
	}
	// StateWriter knows how to write state.
	StateWriter interface {
		WriteState(*State, StateContext) error
	}

	// A StateManager can read and write state
	StateManager interface {
		StateReader
		StateWriter
	}
)
