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
	// R11nResource is a handler factory for r11n handlers.
	R11nResource struct {
		userExtractor
		context ComponentLocator
	}

	// GETR11nHandler handles getting r11ns.
	GETR11nHandler struct {
		WaitForResolution bool
		QueueSet          *sous.R11nQueueSet
		DeploymentID      sous.DeploymentID
		DeploymentIDErr   error
		R11nID            sous.R11nID
		R11nIDErr         error
	}
)

func newR11nResource(ctx ComponentLocator) *R11nResource {
	return &R11nResource{context: ctx}
}

func r11nIDFromRoute(p httprouter.Params) (sous.R11nID, error) {
	ridStr, err := url.PathUnescape(p.ByName("R11nID"))
	if err != nil {
		return "", fmt.Errorf("unescaping path: %s", err)
	}
	return sous.R11nID(ridStr), nil
}

// Get returns a configured GETR11nHandler.
func (mr *R11nResource) Get(_ http.ResponseWriter, r *http.Request, p httprouter.Params) restful.Exchanger {
	did, didErr := deploymentIDFromRoute(p)
	rid, ridErr := r11nIDFromRoute(p)
	wait := r.URL.Query().Get("wait") == "true"
	return &GETR11nHandler{
		DeploymentID:      did,
		DeploymentIDErr:   didErr,
		R11nID:            rid,
		R11nIDErr:         ridErr,
		WaitForResolution: wait,
	}
}

// Exchange returns the targeted r11nResponse and 200 if it exists, other
// non-200 responses otherwise.
func (gmh *GETR11nHandler) Exchange() (interface{}, int) {
	if gmh.DeploymentIDErr != nil {
		return nil, 404
	}
	queues := gmh.QueueSet.Queues()
	queue, ok := queues[gmh.DeploymentID]
	if !ok {
		return deploymentResponse{}, 404
	}
	qr, ok := queue.ByID(gmh.R11nID)
	if !ok {
		return r11nResponse{}, 404
	}
	if gmh.WaitForResolution {
		qr.Rectification.Wait()
	}
	return r11nResponse{
		QueuePosition: qr.Pos,
	}, 200
}

type r11nResponse struct {
	QueuePosition int
}
