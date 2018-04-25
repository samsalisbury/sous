package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// AllDeployQueuesResource describes resources for deployments.
	AllDeployQueuesResource struct {
		context ComponentLocator
	}

	// GETAllDeployQueuesHandler handles GET exchanges for deployments.
	GETAllDeployQueuesHandler struct {
		QueueSet sous.QueueSet
	}
)

func newAllDeployQueuesResource(ctx ComponentLocator) *AllDeployQueuesResource {
	return &AllDeployQueuesResource{context: ctx}
}

// Get returns a configured GETAllDeployQueuesHandler.
func (r *AllDeployQueuesResource) Get(_ *restful.RouteMap, _ logging.LogSink, _ http.ResponseWriter, _ *http.Request, _ httprouter.Params) restful.Exchanger {
	return &GETAllDeployQueuesHandler{
		QueueSet: r.context.QueueSet,
	}
}

// Exchange returns deploymentsResponse representing all queues managed by this
// server instance.
func (h *GETAllDeployQueuesHandler) Exchange() (interface{}, int) {
	data := DeploymentQueuesResponse{Queues: map[string]QueueDesc{}}

	queues := h.QueueSet.Queues()
	for did, q := range queues {
		data.Queues[did.String()] = QueueDesc{
			DeploymentID: did,
			Length:       q.Len(),
		}
	}
	return data, 200
}
