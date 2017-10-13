package actions

import (
	"context"
	"time"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

type pollstatus struct {
	StatusPoller *sous.StatusPoller
}

// GetPollStatus produces an Action to poll the status of a deployment.
func GetPollStatus(di injector, dff config.DeployFilterFlags) Action {
	guardedAdd(di, "DeployFilterFlags", &dff)

	ps := &pollstatus{}
	di.Inject(ps)
	return ps
}

func (ps *pollstatus) Do() error {
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
