package server

import (
	"net"
	"net/http"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
)

// New creates a Sous HTTP server
func New(rm RouteMap, gf func() *graph.SousGraph) *http.Server {
	return &http.Server{
		Handler: BuildRouter(rm, gf),
	}
}

// RunServer starts a server up
func RunServer(v *config.Verbosity, nw, laddr string) error {
	gf := func() *graph.SousGraph {
		g := graph.BuildGraph(os.Stdout, os.Stdout)
		g.Add(v)
		return g
	}
	s := New(SousRouteMap, gf)
	l, err := net.Listen(nw, laddr)
	if err != nil {
		return err
	}
	return s.Serve(l)
}
