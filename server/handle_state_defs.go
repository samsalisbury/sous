package server

import "github.com/opentable/sous/lib"

type (
	StateDefResource struct{}

	StateDefGetHandler struct {
		*sous.State
	}
)

func (sdr *StateDefResource) Get() Exchanger { return &StateDefGetHandler{} }

func (sdg *StateDefGetHandler) Exchange() (interface{}, int) {
	return sdg.State.Defs, 200
}
