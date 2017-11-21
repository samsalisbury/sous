package graph

import (
	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
)

func (di *SousGraph) guardedAdd(guardName string, value interface{}) {
	if di.addGuards[guardName] {
		return
	}
	di.addGuards[guardName] = true
	di.Add(value)
}

// GetUpdate returns an update Action.
func (di *SousGraph) GetUpdate(dff config.DeployFilterFlags, otpl config.OTPLFlags) (actions.Action, error) {
	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("OTPLFlags", &otpl)
	di.guardedAdd("Dryrun", DryrunNeither)

	updateScoop := struct {
		Manifest      TargetManifest
		GDM           CurrentGDM
		Client        HTTPClient
		ResolveFilter *RefinedResolveFilter
		User          sous.User
		LogSink       LogSink
	}{}
	if err := di.Inject(&updateScoop); err != nil {
		return nil, err
	}
	return &actions.Update{
		Manifest:      updateScoop.Manifest.Manifest,
		GDM:           updateScoop.GDM.Deployments,
		Client:        updateScoop.Client.HTTPClient,
		ResolveFilter: (*sous.ResolveFilter)(updateScoop.ResolveFilter),
		User:          updateScoop.User,
		Log:           updateScoop.LogSink.LogSink,
	}, nil
}

// GetRectify produces a rectify Action.
func (di *SousGraph) GetRectify(dryrun string, dff config.DeployFilterFlags) (actions.Action, error) {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	scoop := struct {
		LogSink  LogSink
		Resolver *sous.Resolver
		State    *sous.State
	}{}

	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	return &actions.Rectify{
		Log:      scoop.LogSink.LogSink,
		Resolver: scoop.Resolver,
		State:    scoop.State,
	}, nil
}

// GetPollStatus produces an Action to poll the status of a deployment.
func (di *SousGraph) GetPollStatus(dryrun string, dff config.DeployFilterFlags) (actions.Action, error) {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	scoop := struct {
		StatusPoller *sous.StatusPoller
	}{}

	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}

	return &actions.PollStatus{
		StatusPoller: scoop.StatusPoller,
	}, nil
}
