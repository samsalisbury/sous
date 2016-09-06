package server

import "net/http"

// New creates a Sous HTTP server
func New() *http.Server {
	return &http.Server{
		Handler: BuildRouter(),
	}
}
