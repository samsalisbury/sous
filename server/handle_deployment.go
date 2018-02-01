package server

import (
	"net/http"

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
		QueueSet     *sous.R11nQueueSet
		DeploymentID sous.DeploymentID
	}
)

func newDeploymentResource(ctx ComponentLocator) *DeploymentResource {
	return &DeploymentResource{context: ctx}
}

// Get implements Getable for DeploymentResource.
func (mr *DeploymentResource) Get(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &GETDeploymentHandler{}
}

// Exchange implements restful.Exchanger
func (gmh *GETDeploymentHandler) Exchange() (interface{}, int) {
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
