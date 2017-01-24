package server

import (
	"net/http"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

type (
	// StatusResource encapsulates a status response.
	StatusResource struct {
	}

	// StatusHandler handles requests for status.
	StatusHandler struct {
		GDM          graph.CurrentGDM
		AutoResolver *sous.AutoResolver
		*sous.ResolveFilter
	}

	statusData struct {
		Deployments           []*sous.Deployment
		Completed, InProgress *sous.ResolveStatus
	}
)

// Get implements Getable on StatusResource.
func (*StatusResource) Get() Exchanger { return &StatusHandler{} }

// Exchange implements the Handler interface.
func (h *StatusHandler) Exchange() (interface{}, int) {
	status := statusData{}
	for _, d := range h.GDM.Filter(h.ResolveFilter.FilterDeployment).Snapshot() {
		status.Deployments = append(status.Deployments, d)
	}
	status.Completed, status.InProgress = h.AutoResolver.Statuses()
	return status, http.StatusOK
}
