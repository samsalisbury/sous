package actions

import (
	"context"
	"time"

	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

// PollStatus manages the command to poll the server for status.
type PollStatus struct {
	StatusPoller *sous.StatusPoller
}

// Do implements Action on PollStatus.
func (ps *PollStatus) Do() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	state, err := ps.StatusPoller.Wait(ctx)
	if err != nil {
		return err
	}
	if state != sous.ResolveComplete {
		return errors.Errorf("failed (state is %s)", state)
	}
	return nil
}
