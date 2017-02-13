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

func (sdr *StateDefResource) Get() restful.Exchanger { return &StateDefGetHandler{} }

func (sdg *StateDefGetHandler) Exchange() (interface{}, int) {
	return sdg.State.Defs, 200
}
