package server

import (
	"net/http"

	sous "github.com/opentable/sous/lib"
)

type (
	// PUTSingleDeploymentHandler handles manifests containing single deployment
	// specs. See Exchange method for more details.
	PUTSingleDeploymentHandler struct {
		DeploymentID    sous.DeploymentID
		DeploymentIDErr error
		Body            *singleDeploymentBody
		BodyErr         error
		Header          http.Header
		GDM             *sous.State
		StateWriter     sous.StateWriter
		QueueSet        *sous.R11nQueueSet
		User            sous.User
	}

	// singleDeploymentBody is the response struct returned from handlers
	// of HTTP methods of a SingleDeploymentResource.
	singleDeploymentBody struct {
		Meta           ResponseMeta
		DeploymentID   sous.DeploymentID
		DeploySpec     sous.DeploySpec
		ManifestHeader sous.Manifest
	}

	// ResponseMeta contains metadata to include in API response bodies.
	ResponseMeta struct {
		// Links is a set of links related to a response body.
		Links map[string]string
	}
)

// Exchange triggers a deployment action when receiving
// a Manifest containing a deployment matching DeploymentID that differs
// from the current actual deployment set. It first writes the new
// deployment spec to the GDM.
func (psd *PUTSingleDeploymentHandler) Exchange() (interface{}, int) {
	m, ok := psd.GDM.Manifests.Get(psd.Body.DeploymentID.ManifestID)
	if !ok {
		return psd.Body, 404
	}

	if psd.Body.DeploymentID != psd.DeploymentID {
		return psd.Body, 400
	}

	cluster := psd.Body.DeploymentID.Cluster
	d, ok := m.Deployments[cluster]
	if !ok {
		return psd.Body, 404
	}
	different, _ := psd.Body.DeploySpec.Diff(d)
	if !different {
		return psd.Body, 200
	}

	m.Deployments[cluster] = psd.Body.DeploySpec

	if err := psd.StateWriter.WriteState(psd.GDM, psd.User); err != nil {
		// TODO SS: Don't panic.
		panic(err)
	}

	// The full deployment can only be gotten from the full state, since it
	// relies on State.Defs which is not part of this exchange. Therefore
	// fish it out of the realized GDM returned from .Deployments()
	//
	// TODO SS:
	// Note that this call is expensive, we should come up with a cheaper way
	// to get single deployments.
	deployments, err := psd.GDM.Deployments()
	if err != nil {
		// TODO SS: Don't panic.
		panic(err)
	}
	fullDeployment, ok := deployments.Get(psd.DeploymentID)
	if !ok {
		// TODO SS: Don't panic.
		panic("not ok")
	}

	r := &sous.Rectification{
		Pair: sous.DeployablePair{
			Post: &sous.Deployable{
				Status:     0,
				Deployment: fullDeployment,
			},
			ExecutorData: nil,
		},
	}
	r.Pair.SetID(psd.DeploymentID)

	qr, ok := psd.QueueSet.Push(r)
	if !ok {
		panic("not ok")
	}

	psd.Body.Meta.Links = map[string]string{
		"queuedDeployAction": "/deploy-queue-item?action=" + string(qr.ID),
	}

	return psd.Body, 200
}
