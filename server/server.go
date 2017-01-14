package server

import (
	"net/http"

	"github.com/opentable/sous/graph"
)

// New creates a Sous HTTP server.
func New(laddr string, gf GraphFactory) *http.Server {
	return &http.Server{
		Addr:    laddr,
		Handler: SousRouteMap.BuildRouter(gf),
	}
}

// RunServer starts a server up.
func RunServer(mainGraph *graph.SousGraph, laddr string) error {
	gf := func() Injector {
		g := mainGraph.Clone()
		AddsPerRequest(g)

		return g
	}
	s := New(laddr, gf)
	return s.ListenAndServe()
}
