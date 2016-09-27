package server

import (
	"net/http"
	"runtime/debug"

	"github.com/opentable/sous/lib"
)

type (

	// A StatusHandler processes panics into 500s and other status codes
	StatusHandler struct {
		*sous.LogSet
	}
)

// HandleResponse returns a 500 and logs the error
// It uses the LogSet provided by the graph
func (ph *StatusHandler) HandleResponse(status int, w http.ResponseWriter, data interface{}) {
	w.WriteHeader(status)

	ph.LogSet.Warn.Printf("Responding: %d %s", status, http.StatusText(status))
	if status >= 400 {
		ph.LogSet.Warn.Printf("%+v", data)
	}
	// XXX in a dev mode, print the panic in the response body
	// (normal ops it might leak secure data)
}

// HandlePanic returns a 500 and logs the error
// It uses the LogSet provided by the graph
func (ph *StatusHandler) HandlePanic(w http.ResponseWriter, r *http.Request, recovered interface{}) {
	w.WriteHeader(http.StatusInternalServerError)
	ph.LogSet.Warn.Printf("%+v", recovered)
	ph.LogSet.Warn.Print(string(debug.Stack()))
	ph.LogSet.Warn.Print("Recovered, returned 500")
	// XXX in a dev mode, print the panic in the response body
	// (normal ops it might leak secure data)
}
