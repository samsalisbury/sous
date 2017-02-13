package server

import (
	"net/http"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/restful"
)

// New creates a Sous HTTP server.
func New(laddr string, gf func() restful.Injector) *http.Server {
	return &http.Server{
		Addr:    laddr,
		Handler: SousRouteMap.BuildRouter(gf),
	}
}

// this ensures that certain objects are injected early, so that they'll remain
// the same across requests
type fixedPoints struct {
	*config.Config
}

func ServerHandler(mainGraph *graph.SousGraph) http.Handler {
	mainGraph.Inject(&fixedPoints{})
	gf := func() restful.Injector {
		g := mainGraph.Clone()
		AddsPerRequest(g)

		return g
	}
	return SousRouteMap.BuildRouter(gf)
}

// RunServer starts a server up.
func RunServer(mainGraph *graph.SousGraph, laddr string) error {
	s := &http.Server{
		Addr:    laddr,
		Handler: ServerHandler(mainGraph),
	}
	return s.ListenAndServe()
}
