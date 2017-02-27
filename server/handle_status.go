package server

import (
	"net/http"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// StatusResource encapsulates a status response.
	StatusResource struct {
	}

	// StatusHandler handles requests for status.
	StatusHandler struct {
		AutoResolver  *sous.AutoResolver
		ResolveFilter *sous.ResolveFilter
		Log           *sous.LogSet
	}

	statusData struct {
		Deployments           []*sous.Deployment
		Completed, InProgress *sous.ResolveStatus
	}
)

// Get implements Getable on StatusResource.
func (*StatusResource) Get() restful.Exchanger { return &StatusHandler{} }

// Exchange implements the Handler interface.
func (h *StatusHandler) Exchange() (interface{}, int) {
	status := statusData{}
	h.Log.Vomit.Printf("AutoResolver's GDM: length %d", h.AutoResolver.GDM.Len())
	for _, d := range h.AutoResolver.GDM.Snapshot() {
		h.Log.Vomit.Printf("  AutoResolver's GDM: %#v", d)
	}
	for _, d := range h.AutoResolver.GDM.Filter(h.ResolveFilter.FilterDeployment).Snapshot() {
		h.Log.Vomit.Printf("Status filtered intended deployment: %#v", d)
		status.Deployments = append(status.Deployments, d)
	}
	status.Completed, status.InProgress = h.AutoResolver.Statuses()
	return status, http.StatusOK
}
