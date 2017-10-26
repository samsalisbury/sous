package graph

import (
	"github.com/davecgh/go-spew/spew"
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
func (di *SousGraph) GetUpdate(dff config.DeployFilterFlags, otpl config.OTPLFlags) actions.Action {
	spew.Printf("GetUpdate: %p\n", di)
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
	di.MustInject(&updateScoop)
	return &actions.Update{
		Manifest:      updateScoop.Manifest.Manifest,
		GDM:           updateScoop.GDM.Deployments,
		Client:        updateScoop.Client.HTTPClient,
		ResolveFilter: (*sous.ResolveFilter)(updateScoop.ResolveFilter),
		User:          updateScoop.User,
		Log:           updateScoop.LogSink.LogSink,
	}
}

// GetRectify produces a rectify Action.
func (di *SousGraph) GetRectify(dryrun string, dff config.DeployFilterFlags) actions.Action {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	scoop := struct {
		LogSink  LogSink
		Resolver *sous.Resolver
		State    *sous.State
	}{}

	di.MustInject(&scoop)
	return &actions.Rectify{
		Log:      scoop.LogSink.LogSink,
		Resolver: scoop.Resolver,
		State:    scoop.State,
	}
}

// GetPollStatus produces an Action to poll the status of a deployment.
func (di *SousGraph) GetPollStatus(dryrun string, dff config.DeployFilterFlags) actions.Action {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	scoop := struct {
		StatusPoller *sous.StatusPoller
	}{}

	di.MustInject(&scoop)

	return &actions.PollStatus{
		StatusPoller: scoop.StatusPoller,
	}
}
