package server

import (
	"net/http"

	"github.com/opentable/sous/graph"
	"github.com/samsalisbury/psyringe"
)

// New creates a Sous HTTP server.
func New(laddr string, processGraph, requestGraph Injector) *http.Server {
	return &http.Server{
		Addr:    laddr,
		Handler: SousRouteMap.BuildRouter(processGraph, requestGraph),
	}
}

// BuildRequestGraph derives a per-request scoped DI graph.
func BuildRequestGraph(processGraph Injector) Injector {
	// Get a copy of StateReader to pass down to requestGraph.
	// TODO: Make child scopes a feature of psyringe.
	stateReaderGrabber := &struct{ graph.StateReader }{}
	processGraph.Inject(stateReaderGrabber)
	// Create requestGraph, which is to be cloned for each request.
	return psyringe.New(liveGDM, stateReaderGrabber.StateReader)
}

// RunServer starts a server up.
func RunServer(mainGraph *graph.SousGraph, laddr string) error {
	requestGraph := BuildRequestGraph(mainGraph)
	return New(laddr, mainGraph, requestGraph).ListenAndServe()
}
