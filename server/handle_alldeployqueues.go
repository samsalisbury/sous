package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// AllDeployQueuesResource describes resources for deployments.
	AllDeployQueuesResource struct {
		context ComponentLocator
	}

	// GETAllDeployQueuesHandler handles GET exchanges for deployments.
	GETAllDeployQueuesHandler struct {
		QueueSet *sous.R11nQueueSet
	}
)

func newAllDeployQueuesResource(ctx ComponentLocator) *AllDeployQueuesResource {
	return &AllDeployQueuesResource{context: ctx}
}

// Get returns a configured GETAllDeployQueuesHandler.
func (r *AllDeployQueuesResource) Get(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) restful.Exchanger {
	return &GETAllDeployQueuesHandler{
		QueueSet: r.context.QueueSet,
	}
}

// Exchange returns deploymentsResponse representing all queues managed by this
// server instance.
func (h *GETAllDeployQueuesHandler) Exchange() (interface{}, int) {
	queues := h.QueueSet.Queues()
	m := map[sous.DeploymentID]int{}
	for did, q := range queues {
		m[did] = q.Len()
	}
	return DeploymentQueuesResponse{
		Deployments: m,
	}, 200
}
