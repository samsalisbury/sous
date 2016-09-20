package server

import (
	"net/http"

	"github.com/opentable/sous/lib"
)

type (
	GETManifestHandler struct {
		*sous.State
		*QueryValues
	}
	PUTManifestHandler struct {
		*sous.State
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
