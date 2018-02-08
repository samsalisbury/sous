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

func deploymentIDFromRoute(r *http.Request) (sous.DeploymentID, error) {
	didStr, err := url.QueryUnescape(r.URL.Query().Get("DeploymentID"))
	if err != nil {
		return sous.DeploymentID{}, fmt.Errorf("unescaping query: %s", err)
	}
	did, err := sous.ParseDeploymentID(didStr)
	if err != nil {
		return sous.DeploymentID{}, fmt.Errorf("parsing DeploymentID from query: %s", err)
	}
	return did, nil
}

// Get implements Getable for DeploymentResource.
func (mr *DeploymentResource) Get(_ http.ResponseWriter, r *http.Request, _ httprouter.Params) restful.Exchanger {
	did, didErr := deploymentIDFromRoute(r)
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
