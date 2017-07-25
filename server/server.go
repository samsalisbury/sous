package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/opentable/sous/util/restful"
)

/*
// this ensures that certain objects are injected early, so that they'll remain
// the same across requests
type fixedPoints struct {
	*config.Config
	*graph.StateManager
}
	mainGraph.Inject(&fixedPoints{})
	gf := func() restful.Injector {
		g := mainGraph.Clone()
		AddsPerRequest(g)

		return g
	}

// AddsPerRequest registers items into a SousGraph that need to be fresh per request
func AddsPerRequest(g restful.Injector) {
	g.Add(liveGDM)
	g.Add(getUser)
}

func liveGDM(sr graph.StateReader) (*LiveGDM, error) {
	state, err := graph.NewCurrentState(sr)
	if err != nil {
		return nil, err
	}
	gdm, err := graph.NewCurrentGDM(state)
	if err != nil {
		return nil, err
	}
	// Ignore this error because an empty string etag is acceptable.
	etag, _ := state.GetEtag()
	return &LiveGDM{Etag: etag, Deployments: gdm.Deployments}, nil
}
*/

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

// Handler builds the http.Handler for the Sous server httprouter.
func Handler(gf func() restful.Injector, ls logSet) http.Handler {
	handler := mux(gf, ls)
	addMetrics(handler, ls)
	return handler
}

// Handler builds the http.Handler for the Sous server httprouter.
func ProfilingHandler(gf func() restful.Injector, ls logSet) http.Handler {
	handler := mux(gf, ls)
	addMetrics(handler, ls)
	return handler
}

func mux(gf func() restful.Injector, ls logSet) *http.ServeMux {
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
