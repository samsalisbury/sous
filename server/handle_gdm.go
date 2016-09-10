package server

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

type (
	// GDMHandler is an injectable request handler
	GDMHandler struct {
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
func (h *GDMHandler) Exchange() (interface{}, int) {
	data := gdmWrapper{Deployments: make([]*sous.Deployment, 0)}
	for _, d := range h.GDM.Snapshot() {
		data.Deployments = append(data.Deployments, d)
	}
	return data, 200
}
