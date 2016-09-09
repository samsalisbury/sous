package server

import (
	"encoding/json"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

type (
	// GDMHandler is an injectable request handler
	GDMHandler struct {
		*ResponseWriter
		GDM graph.CurrentGDM
	}

	gdmWrapper struct {
		Deployments []*sous.Deployment
	}
)

// NewGDMHandler is the factory function for GDMHandler
func NewGDMHandler() Exchanger {
	return &GDMHandler{}
}

// Exchange implements the Handler interface
func (h *GDMHandler) Exchange() {
	data := gdmWrapper{Deployments: make([]*sous.Deployment, 0)}
	for _, d := range h.GDM.Snapshot() {
		data.Deployments = append(data.Deployments, d)
	}

	h.ResponseWriter.Header().Add("Content-Type", "application/json")
	e := json.NewEncoder(h.ResponseWriter)
	e.Encode(data)
}
