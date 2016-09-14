package server

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

type (
	// A Exchanger has an Exchange method - which is presumed to write to an
	// injected ResponseWriter
	Exchanger interface {
		Exchange() (interface{}, int)
	}

	// A PanicHandler processes panics into 500s
	PanicHandler struct {
		*sous.LogSet
		*ResponseWriter
	}

	// ResponseWriter wraps the the http.ResponseWriter interface
	// XXX This is a workaround for Psyringe
	ResponseWriter struct {
		http.ResponseWriter
	}

	// An ExchangeFactory builds an Exchanger -
	// they're used to configure the RouteMap
	ExchangeFactory func() Exchanger

	routeEntry struct {
		Name     string
		Methods  []string
		Path     string
		Exchange ExchangeFactory
	}

	// RouteMap is a list of entries for routing
	RouteMap []routeEntry
)

var (
	defaultMethods = []string{"GET", "POST", "PUT", "DELETE"}

	// SousRouteMap is the configuration of route for the application
	SousRouteMap = RouteMap{
		{"gdm", []string{"GET"}, "/gdm", NewGDMHandler},
	}
)

// BuildRouter builds a returns an http.Handler based on some constant configuration
func BuildRouter(rm RouteMap, grf func() *graph.SousGraph) http.Handler {
	r := httprouter.New()

	for _, e := range rm {
		for _, m := range e.Methods {
			r.Handle(m, e.Path, Handling(grf, e.Exchange))
		}
	}

	return r
}

// PathFor constructs a URL which should route back to the named route, with
// supplied parameters
func (rm *RouteMap) PathFor(name string, params map[string]string) (string, error) {
	for _, e := range *rm {
		if e.Name == name {
			// TODO actually handle params
			return e.Path, nil
		}
	}
	return "", errors.Errorf("No route found for name %q", name)
}

func panicsAreFiveHundred(w http.ResponseWriter) {
	if r := recover(); r != nil {
		log.Print("paFH")
		log.Print(r)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Handle recovers from panics, returns a 500 and logs the error
func (ph *PanicHandler) Handle() {
	if r := recover(); r != nil {
		ph.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		ph.LogSet.Warn.Printf("%+v", r)
		ph.LogSet.Warn.Print(string(debug.Stack()))
		ph.LogSet.Warn.Print("Recovered, returned 500")
		// XXX in a dev mode, print the panic in the response body
		// (normal ops it might leak secure data)
	}
}

// Handling (sometimes) updates the local copy of the GDM and formats it
func Handling(graphFac func() *graph.SousGraph, factory ExchangeFactory) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		defer panicsAreFiveHundred(w)

		g := graphFac()
		g.Add(&ResponseWriter{ResponseWriter: w}, r, p)

		ph := &PanicHandler{}
		g.Inject(ph)
		defer ph.Handle()

		h := factory()
		g.Inject(h)

		data, status := h.Exchange()

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(status)

		e := json.NewEncoder(w)
		e.Encode(data)
	}
}
