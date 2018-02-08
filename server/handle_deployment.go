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
	// DeployQueueResource describes resources for single deployments.
	DeployQueueResource struct {
		context ComponentLocator
	}

	// GETDeployQueueHandler handles GET exchanges for single deployments.
	GETDeployQueueHandler struct {
		QueueSet        *sous.R11nQueueSet
		DeploymentID    sous.DeploymentID
		DeploymentIDErr error
	}
)

func newDeployQueueResource(ctx ComponentLocator) *DeployQueueResource {
	return &DeployQueueResource{context: ctx}
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

// Get returns a configured GETDeployQueueHandler.
func (r *DeployQueueResource) Get(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	did, didErr := deploymentIDFromRoute(req)
	return &GETDeployQueueHandler{
		DeploymentID:    did,
		DeploymentIDErr: didErr,
	}
}

// Exchange returns a deployQueueResponse representing a single deploy queue.
func (h *GETDeployQueueHandler) Exchange() (interface{}, int) {
	if h.DeploymentIDErr != nil {
		return nil, 404
	}
	queues := h.QueueSet.Queues()
	queue, ok := queues[h.DeploymentID]
	if !ok {
		return deployQueueResponse{}, 404
	}
	var queued = make([]queuedDeployment, queue.Len())
	for i, qr := range queue.Snapshot() {
		queued[i] = queuedDeployment{
			ID: qr.ID,
		}
	}
	return deployQueueResponse{Queue: queued}, 200
}

type deployQueueResponse struct {
	Queue []queuedDeployment
}

type queuedDeployment struct {
	ID sous.R11nID
}
