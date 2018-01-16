package storage

import (
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

// A DuplexStateManager echoes StateManager operation to a primary StateManager,
// but also ensures that writes occur to the secondary one.
type DuplexStateManager struct {
	primary, secondary sous.StateManager
	log                logging.LogSink
}

// NewDuplexStateManager creates a DuplexStateManager
func NewDuplexStateManager(primary, secondary sous.StateManager, log logging.LogSink) *DuplexStateManager {
	return &DuplexStateManager{
		primary:   primary,
		secondary: secondary,
		log:       log,
	}
}

// ReadState implements StateManager on DuplexStateManager
func (dup *DuplexStateManager) ReadState() (*sous.State, error) {
	user := sous.User{}
	start := time.Now()
	state, err := dup.primary.ReadState()
	if err == nil {
		if err := dup.secondary.WriteState(state, user); err != nil {
			logging.ReportError(dup.log, errors.Wrapf(err, "writing to secondary StateManager"))
		}
	}
	reportReading(dup.log, start, state, err)
	return state, err
}

// WriteState implements StateManager on DuplexStateManager
func (dup *DuplexStateManager) WriteState(state *sous.State, user sous.User) error {
	start := time.Now()
	if err := dup.secondary.WriteState(state, user); err != nil {
		logging.ReportError(dup.log, errors.Wrapf(err, "writing to secondary StateManager"))
	}
	err := dup.primary.WriteState(state, user)
	reportWriting(dup.log, start, state, err)
	return err
}
