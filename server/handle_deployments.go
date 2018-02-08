package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// DeploymentsResource describes resources for deployments.
	DeploymentsResource struct {
		context ComponentLocator
	}

	// GETDeploymentsHandler handles GET exchanges for deployments.
	GETDeploymentsHandler struct {
		QueueSet *sous.R11nQueueSet
	}
)

func newDeploymentsResource(ctx ComponentLocator) *DeploymentsResource {
	return &DeploymentsResource{context: ctx}
}

// Get implements Getable for DeploymentResource.
func (mr *DeploymentsResource) Get(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &GETDeploymentsHandler{}
}

// Exchange implements restful.Exchanger
func (gmh *GETDeploymentsHandler) Exchange() (interface{}, int) {
	queues := gmh.QueueSet.Queues()
	m := map[sous.DeploymentID]int{}
	for did, q := range queues {
		m[did] = q.Len()
	}
	return deploymentsResponse{
		Deployments: m,
	}, 200
}

type deploymentsResponse struct {
	Deployments map[sous.DeploymentID]int
}
