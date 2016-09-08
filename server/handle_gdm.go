package server

import (
	"net/http"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

type (
	// GDMHandler is an injectable request handler
	GDMHandler struct {
		w   http.ResponseWriter
		GDM graph.CurrentGDM
	}

	gdmWrapper struct {
		Deployments []sous.Deployment
	}
)

// NewGDMHandler is the factory function for GDMHandler
func NewGDMHandler() Exchanger {
	return &GDMHandler{}
}

// Execute implements the Handler interface
func (h *GDMHandler) Execute() {
	data := gdmWrapper{}
	for _, d := range h.GDM.Snapshot() {
		data.Deployments = append(data.Deployments, d)
	}

	e := NewEncoder(h.w)
	e.Encode(data)
}
