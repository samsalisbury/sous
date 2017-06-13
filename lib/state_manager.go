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

	// StateContext contains additional data about what is being read or written
	// by a StateManager.
	StateContext struct {
		// User is the user this write is attributed to.
		User User
		// TargetManifestID is the manifest this write is expected to affect.
		// Implementations of StateWriter.WriteState should check that the
		// change being written corresponds with this manifest ID.
		TargetManifestID ManifestID
	}
)
