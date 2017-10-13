package graph

import (
	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
)

func (di *SousGraph) guardedAdd(di injector, guardName string, value interface{}) {
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

	update = &actions.Update{}
	di.Inject(update)
	return update
}

// GetRectify produces a rectify Action.
func (di *SousGraph) GetRectify(dryrun string, srcFlags cli.SourceFlags) actions.Action {
	di.guardedAdd("Dryrun", graph.DryrunOption(dryrun))
	di.guardedAdd("SourceFlags", &srcFlags)

	r := &actions.Rectify{}
	di.Inject(r)
	return r
}

// GetPollStatus produces an Action to poll the status of a deployment.
func (di *SousGraph) GetPollStatus(dff config.DeployFilterFlags) actions.Action {
	di.guardedAdd("DeployFilterFlags", &dff)

	ps := &actions.PollStatus{}
	di.Inject(ps)
	return ps
}
