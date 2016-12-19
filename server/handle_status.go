package server

type (
	// StatusResource encapsulates a status response.
	StatusResource struct{}

	// StatusHandler handles requests for status.
	StatusHandler struct{}
)

// Get implements Getable on StatusResource.
func (gr *StatusResource) Get() Exchanger { return &StatusHandler{} }

// Exchange implements the Handler interface
func (h *StatusHandler) Exchange() (interface{}, int) {
	return nil, 404
}
