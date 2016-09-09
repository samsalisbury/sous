package server

import (
	"net/http"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
)

// New creates a Sous HTTP server
func New(v *config.Verbosity, laddr string) *http.Server {
	return &http.Server{
		Addr:    laddr,
		Handler: NewSousRouter(v),
	}
}

// NewSousRouter builds a router for the Sous server
func NewSousRouter(v *config.Verbosity) http.Handler {
	gf := func() *graph.SousGraph {
		g := graph.BuildGraph(os.Stdout, os.Stdout)
		g.Add(v)
		return g
	}
	return BuildRouter(SousRouteMap, gf)
}

// RunServer starts a server up
func RunServer(v *config.Verbosity, laddr string) error {
	s := New(v, laddr)
	return s.ListenAndServe()
}
