package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/opentable/sous/lib"
)

type (

	// A StatusMiddleware processes panics into 500s and other status codes.
	StatusMiddleware struct {
		*sous.LogSet
	}
)

func (ph *StatusMiddleware) errorBody(status int, rq *http.Request, w io.Writer, data interface{}, err error, stack []byte) {
	gatelatch := os.Getenv("GATELATCH")
	if gatelatch == "" {
		return
	}

	if header := rq.Header.Get("X-Gatelatch"); header != gatelatch {
		ph.LogSet.Warn.Printf("Gatelatch header (%q) didn't match gatelatch env (%s)", gatelatch, header)
		return
	}

	w.Write([]byte(fmt.Sprintf("Error status: %d\n", status)))
	w.Write([]byte(fmt.Sprintf("Data: %#v\n", data)))
	w.Write([]byte(fmt.Sprintf("Error: %+v\n", err)))

	if stack == nil {
		w.Write([]byte("Created stack: \n"))
		w.Write(debug.Stack())
	} else {
		w.Write([]byte("Passed (panic) stack: \n"))
		w.Write(stack)
	}
	return
}

// HandleResponse returns a 500 and logs the error.
// It uses the LogSet provided by the graph.
func (ph *StatusMiddleware) HandleResponse(status int, r *http.Request, w http.ResponseWriter, data interface{}) {
	w.WriteHeader(status)

	ph.LogSet.Warn.Printf("Responding: %d %s: %s %s", status, http.StatusText(status), r.Method, r.URL)
	if status >= 400 {
		ph.LogSet.Warn.Printf("%+v", data)
		ph.errorBody(status, r, w, data, nil, nil)
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
	stack := debug.Stack()
	if ph.LogSet == nil {
		ph.LogSet = &sous.Log
	}
	ph.LogSet.Warn.Printf("%+v", recovered)
	ph.LogSet.Warn.Print(string(stack))
	ph.LogSet.Warn.Print("Recovered, returned 500")
	ph.errorBody(http.StatusInternalServerError, r, w, nil, recovered.(error), stack)
	// XXX in a dev mode, print the panic in the response body
	// (normal ops it might leak secure data)
}
