package server

import (
	"net/http"
	"runtime/debug"

	"github.com/opentable/sous/lib"
)

type (

	// A StatusMiddleware processes panics into 500s and other status codes.
	StatusMiddleware struct {
		*sous.LogSet
	}
)

// HandleResponse returns a 500 and logs the error.
// It uses the LogSet provided by the graph.
func (ph *StatusMiddleware) HandleResponse(status int, r *http.Request, w http.ResponseWriter, data interface{}) {
	w.WriteHeader(status)

	ph.LogSet.Warn.Printf("Responding: %d %s: %s %s", status, http.StatusText(status), r.Method, r.URL)
	if status >= 400 {
		ph.LogSet.Warn.Printf("%+v", data)
	}
	if status >= 200 && status < 300 {
		ph.LogSet.Debug.Printf("%+v", data)
	}
	// XXX in a dev mode, print the panic in the response body
	// (normal ops it might leak secure data)
}

// HandlePanic returns a 500 and logs the error.
// It uses the LogSet provided by the graph.
func (ph *StatusMiddleware) HandlePanic(w http.ResponseWriter, r *http.Request, recovered interface{}) {
	w.WriteHeader(http.StatusInternalServerError)
	ph.LogSet.Warn.Printf("%+v", recovered)
	ph.LogSet.Warn.Print(string(debug.Stack()))
	ph.LogSet.Warn.Print("Recovered, returned 500")
	// XXX in a dev mode, print the panic in the response body
	// (normal ops it might leak secure data)
}
