package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// StateDefResource defines the /defs endpoint
	StateDefResource struct {
		context ComponentLocator
	}

	StateDefGetHandler struct {
		*sous.State
	}
)

func newStateDefResource(ctx ComponentLocator) *StateDefResource {
	return &StateDefResource{context: ctx}
}

// Get implements restful.Getter on StateDefResource (and therefore makes it
// handle GET requests.)
func (sdr *StateDefResource) Get(http.ResponseWriter, *http.Request, httprouter.Params) restful.Exchanger {
	return &StateDefGetHandler{
		State: sdr.context.LiveState(),
	}
}

// Exchange implements restful.Exchanger on StateDefGetHandler.
func (sdg *StateDefGetHandler) Exchange() (interface{}, int) {
	return sdg.State.Defs, 200
}
