package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// DeploymentResource describes resources for single deployments.
	DeploymentResource struct {
		userExtractor
		context ComponentLocator
	}

	// GETDeploymentHandler handles GET exchanges for single deployments.
	GETDeploymentHandler struct {
		QueueSet        *sous.R11nQueueSet
		DeploymentID    sous.DeploymentID
		DeploymentIDErr error
	}
)

func newDeploymentResource(ctx ComponentLocator) *DeploymentResource {
	return &DeploymentResource{context: ctx}
}

func deploymentIDFromRoute(p httprouter.Params) (sous.DeploymentID, error) {
	didStr, err := url.PathUnescape(p.ByName("DeploymentID"))
	if err != nil {
		return sous.DeploymentID{}, fmt.Errorf("unescaping path: %s", err)
	}
	did, err := sous.ParseDeploymentID(didStr)
	if err != nil {
		return sous.DeploymentID{}, fmt.Errorf("parsing deployment ID from path: %s", err)
	}
	return did, nil
}

// Get implements Getable for DeploymentResource.
func (mr *DeploymentResource) Get(_ http.ResponseWriter, _ *http.Request, p httprouter.Params) restful.Exchanger {
	did, didErr := deploymentIDFromRoute(p)
	return &GETDeploymentHandler{
		DeploymentID:    did,
		DeploymentIDErr: didErr,
	}
}

// Exchange implements restful.Exchanger
func (gmh *GETDeploymentHandler) Exchange() (interface{}, int) {
	if gmh.DeploymentIDErr != nil {
		return nil, 404
	}
	queues := gmh.QueueSet.Queues()
	queue, ok := queues[gmh.DeploymentID]
	if !ok {
		return deploymentResponse{}, 404
	}
	var queued = make([]queuedDeployment, queue.Len())
	for i, qr := range queue.Snapshot() {
		queued[i] = queuedDeployment{
			ID: qr.ID,
		}
	}
	return deploymentResponse{Queue: queued}, 200
}

type deploymentResponse struct {
	Queue []queuedDeployment
}

type queuedDeployment struct {
	ID sous.R11nID
}
