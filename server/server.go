package server

import (
	"bytes"
	"net/http"
	"os"

	"github.com/opentable/sous/config"
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
func RunServer(v *config.Verbosity, laddr string) error {
	mainGraph := graph.BuildGraph(&bytes.Buffer{}, os.Stdout, os.Stdout)
	gf := func() Injector {
		g := mainGraph.Clone()
		g.Add(v)
		return g
	}
	s := New(laddr, gf)
	return s.ListenAndServe()
}
