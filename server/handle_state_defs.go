package server

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	StateDefResource struct{}

	StateDefGetHandler struct {
		*sous.State
	}
)

// Get implements restful.Getter on StateDefResource (and therefore makes it
// handle GET requests.)
func (sdr *StateDefResource) Get() restful.Exchanger { return &StateDefGetHandler{} }

// Exchange implements restful.Exchanger on StateDefGetHandler.
func (sdg *StateDefGetHandler) Exchange() (interface{}, int) {
	return sdg.State.Defs, 200
}
