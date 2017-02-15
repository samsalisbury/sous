package server

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// GDMResource is the resource for the GDM
	GDMResource struct{}

	// GDMHandler is an injectable request handler
	GDMHandler struct {
		GDM *LiveGDM
	}

	gdmWrapper struct {
		Deployments []*sous.Deployment
	}
)

// Get implements Getable on GDMResource
func (gr *GDMResource) Get() restful.Exchanger { return &GDMHandler{} }

// Exchange implements the Handler interface
func (h *GDMHandler) Exchange() (interface{}, int) {
	sous.Log.Debug.Print(h.GDM)
	data := gdmWrapper{Deployments: make([]*sous.Deployment, 0)}
	for _, d := range h.GDM.Snapshot() {
		data.Deployments = append(data.Deployments, d)
	}
	return data, 200
}
