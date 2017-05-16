package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/restful"
)

/*
// New creates a Sous HTTP server.
func New(laddr string, gf func() restful.Injector) *http.Server {
	return &http.Server{
		Addr:    laddr,
		Handler: SousRouteMap.BuildRouter(gf),
	}
}
*/

// this ensures that certain objects are injected early, so that they'll remain
// the same across requests
type fixedPoints struct {
	*config.Config
}

// Handler builds the http.Handler for the Sous server httprouter.
func Handler(mainGraph *graph.SousGraph) http.Handler {
	mainGraph.Inject(&fixedPoints{})
	gf := func() restful.Injector {
		g := mainGraph.Clone()
		AddsPerRequest(g)

		return g
	}
	return SousRouteMap.BuildRouter(gf)
}

// Run starts a server up.
func Run(mainGraph *graph.SousGraph, laddr string) error {
	s := &http.Server{
		Addr:    laddr,
		Handler: Handler(mainGraph),
	}
	return s.ListenAndServe()
}

func profilingHandler(mainGraph *graph.SousGraph) http.Handler {
	handler := http.NewServeMux()
	handler.Handle("/", Handler(mainGraph))

	handler.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	handler.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	handler.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	handler.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	handler.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	return handler
}

// RunWithProfiling mixes in the pprof handlers so that we can return profiles
func RunWithProfiling(mainGraph *graph.SousGraph, laddr string) error {
	Handler(mainGraph)
	s := &http.Server{
		Addr:    laddr,
		Handler: profilingHandler(mainGraph),
	}
	return s.ListenAndServe()
}
