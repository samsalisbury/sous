package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/restful"
)

// this ensures that certain objects are injected early, so that they'll remain
// the same across requests
type fixedPoints struct {
	*config.Config
	*graph.StateManager
}

type logSet interface {
	Vomitf(format string, a ...interface{})
	Debugf(format string, a ...interface{})
	Warnf(format string, a ...interface{})
}

// Handler builds the http.Handler for the Sous server httprouter.
func Handler(mainGraph *graph.SousGraph, ls logSet) http.Handler {
	mainGraph.Inject(&fixedPoints{})
	gf := func() restful.Injector {
		g := mainGraph.Clone()
		AddsPerRequest(g)

		return g
	}
	return SousRouteMap.BuildRouter(gf, ls)
}

// Run starts a server up.
func Run(mainGraph *graph.SousGraph, laddr string, ls logSet) error {
	s := &http.Server{
		Addr:    laddr,
		Handler: Handler(mainGraph, ls),
	}
	return s.ListenAndServe()
}

func profilingHandler(mainGraph *graph.SousGraph, ls logSet) http.Handler {
	handler := http.NewServeMux()
	handler.Handle("/", Handler(mainGraph, ls))

	handler.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	handler.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	handler.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	handler.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	handler.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	return handler
}

// RunWithProfiling mixes in the pprof handlers so that we can return profiles
func RunWithProfiling(mainGraph *graph.SousGraph, laddr string, ls logSet) error {
	s := &http.Server{
		Addr:    laddr,
		Handler: profilingHandler(mainGraph, ls),
	}
	return s.ListenAndServe()
}
