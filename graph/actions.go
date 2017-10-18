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
func (di *SousGraph) GetUpdate(dff config.DeployFilterFlags, otpl config.OTPLFlags) actions.Action {
	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("OTPLFlags", &otpl)
	di.guardedAdd("Dryrun", DryrunNeither)

	updateScoop := struct {
		Manifest      TargetManifest
		GDM           CurrentGDM
		Client        HTTPClient
		ResolveFilter *RefinedResolveFilter
		User          sous.User
	}{}
	di.MustInject(&updateScoop)
	return &actions.Update{
		Manifest:      updateScoop.Manifest.Manifest,
		GDM:           updateScoop.GDM.Deployments,
		Client:        updateScoop.Client.HTTPClient,
		ResolveFilter: (*sous.ResolveFilter)(updateScoop.ResolveFilter),
		User:          updateScoop.User,
	}
}

// GetRectify produces a rectify Action.
func (di *SousGraph) GetRectify(dryrun string, dff config.DeployFilterFlags) actions.Action {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	r := &actions.Rectify{}
	di.MustInject(r)
	return r
}

// GetPollStatus produces an Action to poll the status of a deployment.
func (di *SousGraph) GetPollStatus(dryrun string, dff config.DeployFilterFlags) actions.Action {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	ps := &actions.PollStatus{}
	di.MustInject(ps)
	return ps
}
