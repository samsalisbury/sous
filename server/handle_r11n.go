package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/dto"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// R11nResource is a handler factory for r11n handlers.
	R11nResource struct {
		context ComponentLocator
	}

	// GETR11nHandler handles getting r11ns.
	GETR11nHandler struct {
		WaitForResolution bool
		QueueSet          sous.QueueSet
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
	ridStr, err := url.QueryUnescape(r.URL.Query().Get("action"))
	if err != nil {
		return "", fmt.Errorf("unescaping query: %s", err)
	}
	return sous.R11nID(ridStr), nil
}

// Get returns a configured GETR11nHandler.
func (r *R11nResource) Get(_ *restful.RouteMap, _ logging.LogSink, _ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	did, didErr := deploymentIDFromValues(restful.QueryValues{Values: req.URL.Query()})
	rid, ridErr := r11nIDFromRoute(req)
	wait := req.URL.Query().Get("wait") == "true"
	return &GETR11nHandler{
		QueueSet:          r.context.QueueSet,
		WaitForResolution: wait,
		DeploymentID:      did,
		DeploymentIDErr:   didErr,
		R11nID:            rid,
		R11nIDErr:         ridErr,
	}
}

// Exchange returns the targeted r11nResponse and 200 if it exists, other
// non-200 responses otherwise.
func (h *GETR11nHandler) Exchange() (interface{}, int) {
	if h.DeploymentIDErr != nil {
		return nil, http.StatusNotFound
	}
	// Note that all queries and waiting should be done using the QueueSet
	// itself, not the rectification.
	if h.WaitForResolution {
		_, ok := h.QueueSet.Wait(h.DeploymentID, h.R11nID)
		if !ok {
			return fmt.Sprintf("Deploy action %q not found in queue for %q.",
				h.R11nID, h.DeploymentID), http.StatusNotFound
		}
	}
	queues := h.QueueSet.Queues()
	queue, ok := queues[h.DeploymentID]
	if !ok {
		return fmt.Sprintf("Nothing queued for %q.", h.DeploymentID),
			http.StatusNotFound
	}
	qr, ok := queue.ByID(h.R11nID)
	if !ok {
		return fmt.Sprintf("Deploy action %q not found in queue for %q.",
			h.R11nID, h.DeploymentID), http.StatusNotFound
	}

	// XXX Should this be part of the ByID contract?
	// Specifically, the Resolution field would need to be *DiffResolution
	rez := &qr.Rectification.Resolution
	if qr.Pos >= 0 {
		rez = nil
	}

	return dto.R11nResponse{
		QueuePosition: qr.Pos,
		Resolution:    rez,
	}, http.StatusOK
}

/*
type r11nResponse struct {
	QueuePosition int
	// Pointer here is just to allow nil which is a clearer indication of
	// "nothing to see here" than a JSON-marshalled zero value would be.
	Resolution *sous.DiffResolution
}
*/
