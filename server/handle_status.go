package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// StatusResource encapsulates a status response.
	StatusResource struct {
		context ComponentLocator
	}

	// StatusHandler handles requests for status.
	StatusHandler struct {
		AutoResolver *sous.AutoResolver
		*sous.ResolveFilter
	}

	statusData struct {
		Deployments           []*sous.Deployment
		Completed, InProgress *sous.ResolveStatus
	}
)

func newStatusResource(ctx ComponentLocator) *StatusResource {
	return &StatusResource{context: ctx}
}

// Get implements Getable on StatusResource.
func (sr *StatusResource) Get(*restful.RouteMap, http.ResponseWriter, *http.Request, httprouter.Params) restful.Exchanger {
	return &StatusHandler{
		AutoResolver:  sr.context.AutoResolver,
		ResolveFilter: sr.context.ResolveFilter,
	}
}

// Exchange implements the Handler interface.
func (h *StatusHandler) Exchange() (interface{}, int) {
	status := statusData{}
	for _, d := range h.AutoResolver.GDM.Filter(h.ResolveFilter.FilterDeployment).Snapshot() {
		status.Deployments = append(status.Deployments, d)
	}
	status.Completed, status.InProgress = h.AutoResolver.Statuses()
	return status, http.StatusOK
}
