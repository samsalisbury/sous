package server

import (
	"net/http"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
)

// New creates a Sous HTTP server
func New(laddr string, gf GraphFactory) *http.Server {
	return &http.Server{
		Addr:    laddr,
		Handler: SousRouteMap.BuildRouter(gf),
	}
}

// RunServer starts a server up
func RunServer(v *config.Verbosity, laddr string) error {
	gf := func() Injector {
		g := graph.BuildGraph(os.Stdout, os.Stdout)
		g.Add(v)
		return g
	}
	s := New(laddr, gf)
	return s.ListenAndServe()
}
