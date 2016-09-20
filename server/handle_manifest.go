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

To look up a manifest, we need a manifestID:
ManifestID{
	SourceLocation{
		Repo
		Offset
	}
	Flavor
}
Which is a SourceLocation
*/

func manifestIDFromValues(qv *QueryValues) (ManifestID, error) {
	repo := qv.Get("repo")
	offset := qv.Get("offset")
	flavor := qv.Get("flavor")
}

// Exchange implements Exchanger
func (gmh *GETManifestHandler) Exchange() (interface{}, int) {
	mid, err := manifestIDFromValues(gmh.QueryValues)
	if err != nil {
		return nil, http.StatusNotFound
	}
	m, err := gmh.State.Manifests.Get(mid)
	if err != nil {
		return nil, http.StatusNotFound
	}
	return m, http.StatusOK
}

// Exchange implements Exchanger
func (pmh *PUTManifestHandler) Exchange() (interface{}, int) {
	mid, err := manifestIDFromValues(pmh.QueryValues)

}
