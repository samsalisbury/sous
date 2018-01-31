package storage

import (
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

// A LogOnlyStateManager trivially implements StateManager, simply logging the
// actions requested of it.
type LogOnlyStateManager struct {
	log logging.LogSink
}

// NewLogOnlyStateManager returns a new LogOnlyStateManager
func NewLogOnlyStateManager(log logging.LogSink) *LogOnlyStateManager {
	return &LogOnlyStateManager{log: log}
}

// ReadState implements StateManager on LogOnlyStateManager
func (losm LogOnlyStateManager) ReadState() (*sous.State, error) {
	state := sous.NewState()
	reportReading(losm.log, time.Now(), state, nil)
	return state, nil
}

// WriteState implements StateManager on LogOnlyStateManager
func (losm LogOnlyStateManager) WriteState(state *sous.State, _ sous.User) error {
	reportWriting(losm.log, time.Now(), state, nil)
	return nil
}
