package server

import (
	"net/http"

	"github.com/opentable/sous/graph"
)

// New creates a Sous HTTP server
func New(rm RouteMap, gf func() *graph.SousGraph) *http.Server {
	return &http.Server{
		Handler: BuildRouter(rm, gf),
	}
}
