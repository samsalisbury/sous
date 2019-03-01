package server

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// StateDefResource defines the /defs endpoint
	StateDefResource struct {
		userExtractor
		context ComponentLocator
	}

	// StateDefGetHandler handles GET /defs.
	StateDefGetHandler struct {
		*sous.State
	}

	// StateDefPutHandler handles PUT /defs.
	StateDefPutHandler struct {
		sous.StateManager
		req  *http.Request
		user ClientUser
	}
)

func newStateDefResource(ctx ComponentLocator) *StateDefResource {
	return &StateDefResource{context: ctx}
}

// Get implements restful.Getter on StateDefResource (and therefore makes it
// handle GET requests.)
func (sdr *StateDefResource) Get(*restful.RouteMap, logging.LogSink, http.ResponseWriter, *http.Request, httprouter.Params) restful.Exchanger {
	return &StateDefGetHandler{
		State: sdr.context.liveState(),
	}
}

// Put implements restful.Putter on StateDefResource (and therefore makes it
// handle PUT requests.)
func (sdr *StateDefResource) Put(_ *restful.RouteMap, _ logging.LogSink, _ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &StateDefPutHandler{
		StateManager: sdr.context.StateManager,
		req:          req,
		user:         sdr.GetUser(req),
	}
}

// Exchange implements restful.Exchanger on StateDefGetHandler.
func (sdg *StateDefGetHandler) Exchange() (interface{}, int) {
	if sdg.State == nil {
		// State can be nil in the case of  errors reading it.
		return "Unable to read state, please see logs.", 500
	}
	return sdg.State.Defs, 200
}

// Exchange implements restful.Exchanger on StateDefGetHandler.
func (sdp *StateDefPutHandler) Exchange() (interface{}, int) {
	defs := sous.Defs{}
	dec := json.NewDecoder(sdp.req.Body)
	dec.Decode(&defs)

	state, err := sdp.StateManager.ReadState()
	if err != nil {
		msg := "Error loading state from storage"
		return msg, http.StatusInternalServerError
	}

	state.Defs = defs
	err = sdp.StateManager.WriteState(state, sous.User(sdp.user))
	if err != nil {
		msg := "Error recording state to storage"
		return msg, http.StatusInternalServerError
	}

	return nil, 204
}
