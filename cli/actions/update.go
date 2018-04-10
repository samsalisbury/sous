package actions

import (
	"fmt"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
)

// Update is the command description for `sous update`
type Update struct {
	Manifest         *sous.Manifest
	GDM              sous.Deployments
	HTTPStateManager *sous.HTTPStateManager
	ResolveFilter    *sous.ResolveFilter
	User             sous.User
	Log              logging.LogSink
}

// Do performs the appropriate update, returning nil on success.
func (u *Update) Do() error {
	mid := u.Manifest.ID()

	sid, err := u.ResolveFilter.SourceID(mid)
	if err != nil {
		return err
	}
	did, err := u.ResolveFilter.DeploymentID(mid)
	if err != nil {
		return err
	}

	gdm, err := updateRetryLoop(u.Log, u.HTTPStateManager, sid, did, u.User)
	if err != nil {
		return err
	}

	// we update the in-memory GDM so that we can poll based on it.
	for k, d := range gdm.Snapshot() {
		u.GDM.Set(k, d)
	}
	return nil
}

// If multiple updates are attempted at once for different clusters, there's
// the possibility that they will collide in their updates, either interleaving
// their GDM retrieve/manifest update operations, or the git pull/push
// server-side. In this case, the disappointed `sous update` should retry, up
// to the number of times of manifests there are defined for this
// SourceLocation
func updateRetryLoop(ls logging.LogSink,
	sm *sous.HTTPStateManager,
	sid sous.SourceID,
	did sous.DeploymentID,
	user sous.User) (sous.Deployments, error) {

	tryLimit := 2

	mid := did.ManifestID

	start := time.Now()

	for tries := 0; tries < tryLimit; tries++ {
		logging.NewDeliver(ls, newUpdateBeginMessage(tries, sid, did, user, start))

		state, err := sm.ReadState()
		if err != nil {
			return sous.NewDeployments(), err
		}
		manifest, ok := state.Manifests.Get(mid)
		if !ok {
			err := fmt.Errorf("no manifest found for %q - try 'sous init' first", mid)
			logging.NewDeliver(ls, newUpdateErrorMessage(tries, sid, did, user, start, err))
			return sous.NewDeployments(), err
		}

		tryLimit = len(manifest.Deployments)

		gdm, err := state.Deployments()
		if err != nil {
			logging.NewDeliver(ls, newUpdateErrorMessage(tries, sid, did, user, start, err))
			return sous.NewDeployments(), err
		}

		if err := updateState(state, gdm, sid, did); err != nil {
			logging.NewDeliver(ls, newUpdateErrorMessage(tries, sid, did, user, start, err))
			return sous.NewDeployments(), err
		}
		if err := sm.WriteState(state, user); err != nil {
			if !restful.Retryable(err) {
				logging.NewDeliver(ls, newUpdateErrorMessage(tries, sid, did, user, start, err))
				return sous.NewDeployments(), err
			}

			continue
		}

		logging.NewDeliver(ls, newUpdateSuccessMessage(tries, sid, did, manifest, user, start))
		return gdm, nil
	}

	err := errors.Errorf("Tried %d to update %v - %v", tryLimit, sid, did)
	logging.NewDeliver(ls, newUpdateErrorMessage(tryLimit, sid, did, user, start, err))
	return sous.NewDeployments(), err
}

func updateState(s *sous.State, gdm sous.Deployments, sid sous.SourceID, did sous.DeploymentID) error {
	deployment, ok := gdm.Get(did)
	if !ok {
		deployment = &sous.Deployment{}
	}

	deployment.SourceID = sid
	deployment.ClusterName = did.Cluster

	return s.UpdateDeployments(deployment)
}
