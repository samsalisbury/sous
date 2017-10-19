package server

import (
	"net/http"

	"github.com/davecgh/go-spew/spew"
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
	spew.Printf("%p\n\n", ctx)
	return &StateDefResource{context: ctx}
}

// Get implements restful.Getter on StateDefResource (and therefore makes it
// handle GET requests.)
func (sdr *StateDefResource) Get(http.ResponseWriter, *http.Request, httprouter.Params) restful.Exchanger {
	spew.Printf("%p\n\n", sdr.context)
	return &StateDefGetHandler{
		State: sdr.context.liveState(),
	}
}

// Exchange implements restful.Exchanger on StateDefGetHandler.
func (sdg *StateDefGetHandler) Exchange() (interface{}, int) {
	return sdg.State.Defs, 200
}
