package server

import "net/http"

type (
	// GDMHandler is an injectable request handler
	GDMHandler struct {
		w http.ResponseWriter
	}
)

func NewGDMHandler() Exchanger {
	return &GDMHandler{}
}

// Execute implements the Handler interface
func (h *GDMHandler) Execute() {
	fmt.FPrintln(h.w, "Coming soon: the Sous GDM")
}
