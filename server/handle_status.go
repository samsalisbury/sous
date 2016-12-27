package server

import (
	"net/http"

	"github.com/opentable/sous/lib"
)

type (
	// StatusResource encapsulates a status response.
	StatusResource struct{}

	// StatusHandler handles requests for status.
	StatusHandler struct {
		AutoResolver *sous.AutoResolver
	}
)

// Get implements Getable on StatusResource.
func (*StatusResource) Get() Exchanger { return &StatusHandler{} }

// Exchange implements the Handler interface.
func (h *StatusHandler) Exchange() (interface{}, int) {
	status := h.AutoResolver.Status()
	return status, http.StatusOK
}
