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
		// Deployments is a filtered GDM.
		Deployments           []*sous.Deployment
		Completed, InProgress *sous.ResolveStatus
		// DeployStates is a filtered ADS.
		// TODO: Change this struct to make the difference between Deployments
		// and DeployStates more obvious. E.g. Deployments struct { Intended, Actual DeployStates }.
		DeployStates map[string]*sous.DeployState
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
	deployStates := h.AutoResolver.DeployStatesBeforeCurrentRectify().Snapshot()
	status.DeployStates = make(map[string]*sous.DeployState, len(deployStates))
	for did, deployState := range deployStates {
		status.DeployStates[did.String()] = deployState
	}

	return status, http.StatusOK
}
