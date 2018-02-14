package server

import (
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// SingleDeploymentResource describes resources for single deployments.
	SingleDeploymentResource struct {
		context ComponentLocator
	}

	// PUTSingleDeploymentHandler handles PUT exchanges for single deployments.
	PUTSingleDeploymentHandler struct {
		QueueSet        *sous.R11nQueueSet
		RequestBody     io.ReadCloser
		DeploymentID    sous.DeploymentID
		DeploymentIDErr error
	}
)

func newSingleDeploymentResource(ctx ComponentLocator) *SingleDeploymentResource {
	return &SingleDeploymentResource{context: ctx}
}

// Put returns a configured PUTSingleDeploymentHandler.
func (r *SingleDeploymentResource) Put(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	did, didErr := deploymentIDFromRoute(req)
	return &PUTSingleDeploymentHandler{
		RequestBody:     req.Body,
		DeploymentID:    did,
		DeploymentIDErr: didErr,
	}
}

// Exchange parses a partial manifest to generate a diff and enqueues a
// rectification.
func (h *PUTSingleDeploymentHandler) Exchange() (interface{}, int) {
	if h.DeploymentIDErr != nil {
		return nil, 404
	}
	return nil, 500
}
