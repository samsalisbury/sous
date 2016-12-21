package server

import (
	"bytes"
	"net/http"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

// New creates a Sous HTTP server.
func New(laddr string, gf GraphFactory) *http.Server {
	return &http.Server{
		Addr:    laddr,
		Handler: SousRouteMap.BuildRouter(gf),
	}
}

// RunServer starts a server up.
func RunServer(v *config.Verbosity, laddr string, ar *sous.AutoResolver) error {
	gf := func() Injector {
		g := graph.BuildGraph(&bytes.Buffer{}, os.Stdout, os.Stdout)
		g.Add(v)
		g.Add(ar)
		return g
	}
	s := New(laddr, gf)
	return s.ListenAndServe()
}
