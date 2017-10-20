package actions

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
)

// SousUpdate is the command description for `sous update`
type Update struct {
	Manifest      *sous.Manifest
	GDM           sous.Deployments
	Client        restful.HTTPClient
	ResolveFilter *sous.ResolveFilter
	User          sous.User
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

	gdm, err := updateRetryLoop(u.Client, sid, did, u.User)
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
// their GDM retreive/manifest update operations, or the git pull/push
// server-side. In this case, the disappointed `sous update` should retry, up
// to the number of times of manifests there are defined for this
// SourceLocation
func updateRetryLoop(cl restful.HTTPClient, sid sous.SourceID, did sous.DeploymentID, user sous.User) (sous.Deployments, error) {

	sm := sous.NewHTTPStateManager(cl)

	tryLimit := 2

	mid := did.ManifestID

	for tries := 0; tries < tryLimit; tries++ {
		state, err := sm.ReadState()
		if err != nil {
			return sous.NewDeployments(), err
		}
		manifest, ok := state.Manifests.Get(mid)
		if !ok {
			return sous.NewDeployments(), cmdr.UsageErrorf("No manifest found for %q - try 'sous init' first.", mid)
		}

		tryLimit = len(manifest.Deployments)

		gdm, err := state.Deployments()
		if err != nil {
			return sous.NewDeployments(), err
		}

		if err := updateState(state, gdm, sid, did); err != nil {
			return sous.NewDeployments(), err
		}
		if err := sm.WriteState(state, user); err != nil {
			if !restful.Retryable(err) {
				return sous.NewDeployments(), err
			}

			continue
		}

		return gdm, nil
	}
	return sous.NewDeployments(), errors.Errorf("Tried %d to update %v - %v", tryLimit, sid, did)
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
