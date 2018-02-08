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

func r11nIDFromRoute(r *http.Request) (sous.R11nID, error) {
	ridStr, err := url.QueryUnescape(r.URL.Query().Get("R11nID"))
	if err != nil {
		return "", fmt.Errorf("unescaping query: %s", err)
	}
	return sous.R11nID(ridStr), nil
}

// Get returns a configured GETR11nHandler.
func (mr *R11nResource) Get(_ http.ResponseWriter, r *http.Request, _ httprouter.Params) restful.Exchanger {
	did, didErr := deploymentIDFromRoute(r)
	rid, ridErr := r11nIDFromRoute(r)
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
		return nil, http.StatusNotFound
	}
	// Note that all queries and waiting should be done using the QueueSet
	// itself, not the rectification.
	if gmh.WaitForResolution {
		r, ok := gmh.QueueSet.Wait(gmh.DeploymentID, gmh.R11nID)
		if !ok {
			return r11nResponse{}, http.StatusNotFound
		}
		return r11nResponse{
			Resolution: &r,
		}, http.StatusOK
	}
	queues := gmh.QueueSet.Queues()
	queue, ok := queues[gmh.DeploymentID]
	if !ok {
		return deploymentResponse{}, http.StatusNotFound
	}
	qr, ok := queue.ByID(gmh.R11nID)
	if !ok {
		return r11nResponse{}, http.StatusNotFound
	}
	return r11nResponse{
		QueuePosition: qr.Pos,
	}, http.StatusOK
}

type r11nResponse struct {
	QueuePosition int
	// Pointer here is just to allow nil which is a clearer indication of
	// "nothing to see here" than a JSON-marshalled zero value would be.
	Resolution *sous.DiffResolution
}
