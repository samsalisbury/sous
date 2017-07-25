package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/hydrogen18/memlistener"
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
	ExpHandler() http.Handler
}

// Run starts a server up.
func Run(laddr string, handler http.Handler) error {
	s := &http.Server{Addr: laddr, Handler: handler}
	return s.ListenAndServe()
}

// InMemory creates an in memory server.
func InMemory(handler http.Handler) *memlistener.MemoryServer {
	ms := memlistener.NewInMemoryServer(handler)
}

// Handler builds the http.Handler for the Sous server httprouter.
func Handler(mainGraph *graph.SousGraph, ls logSet) http.Handler {
	handler := mux(mainGraph, ls)
	addMetrics(handler, ls)
	return handler
}

// Handler builds the http.Handler for the Sous server httprouter.
func ProfilingHandler(mainGraph *graph.SousGraph, ls logSet) http.Handler {
	handler := mux(mainGraph, ls)
	addMetrics(handler, ls)
	return handler
}

func mux(mainGraph *graph.SousGraph, ls logSet) *http.ServeMux {
	mainGraph.Inject(&fixedPoints{})
	gf := func() restful.Injector {
		g := mainGraph.Clone()
		AddsPerRequest(g)

		return g
	}
	router := SousRouteMap.BuildRouter(gf, ls)

	handler := http.NewServeMux()
	handler.Handle("/", router)
	return handler
}

func addMetrics(handler *http.ServeMux, ls logSet) {
	handler.Handle("/debug/metrics", ls.ExpHandler())
}

func addProfiling(handler *http.ServeMux) {
	handler.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	handler.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	handler.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	handler.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	handler.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
}
