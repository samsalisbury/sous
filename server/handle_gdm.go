package server

import (
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

type (
	// GDMResource is the resource for the GDM
	GDMResource struct{}

	// GDMHandler is an injectable request handler
	GDMHandler struct {
		GDM graph.CurrentGDM
	}

	gdmWrapper struct {
		Deployments []*sous.Deployment
	}
)

// Get implements Getable on GDMResource
func (gr *GDMResource) Get() Exchanger { return &GDMHandler{} }

// Exchange implements the Handler interface
func (h *GDMHandler) Exchange() (interface{}, int) {
	data := gdmWrapper{Deployments: make([]*sous.Deployment, 0)}
	for _, d := range h.GDM.Snapshot() {
		data.Deployments = append(data.Deployments, d)
	}
	return data, 200
}
