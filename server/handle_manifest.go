package server

import (
	"net/http"

	"github.com/opentable/sous/graph"
)

type (
	GETManifestHandler struct {
		GDM graph.CurrentGDM
		*QueryValues
	}
	PUTManifestHandler struct {
		GDM graph.CurrentGDM
		*http.Request
		*QueryValues
	}
)

/*
To recap:

To look up a manifest, ma
*/

// Exchange implements Exchanger
func (gmh *GETManifestHandler) Exchange() (interface{}, int) {
	repo := gmh.Get("repo")
	offset := gmh.Get("offset")
}

// Exchange implements Exchanger
func (pmh *PUTManifestHandler) Exchange() (interface{}, int) {
	repo := gmh.Get("repo")
	offset := gmh.Get("offset")

}
