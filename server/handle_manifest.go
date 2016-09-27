package server

import (
	"encoding/json"
	"net/http"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
)

type (
	// ManifestResource describes resources for manifests
	ManifestResource struct{}

	// GETManifestHandler handles GET exchanges for manifests
	GETManifestHandler struct {
		*sous.State
		*QueryValues
	}

	// PUTManifestHandler handles PUT exchanges for manifests
	PUTManifestHandler struct {
		*sous.State
		*http.Request
		*QueryValues
		StateWriter graph.LocalStateWriter
	}
)

// Get implements Getable for ManifestResource
func (mr *ManifestResource) Get() Exchanger { return &GETManifestHandler{} }

// Put implements Putable for ManifestResource
func (mr *ManifestResource) Put() Exchanger { return &PUTManifestHandler{} }

// Exchange implements Exchanger
func (gmh *GETManifestHandler) Exchange() (interface{}, int) {
	mid, err := manifestIDFromValues(gmh.QueryValues)
	if err != nil {
		return err, http.StatusNotFound
	}
	m, there := gmh.State.Manifests.Get(mid)
	if !there {
		return nil, http.StatusNotFound
	}
	return m, http.StatusOK
}

// Exchange implements Exchanger
func (pmh *PUTManifestHandler) Exchange() (interface{}, int) {
	mid, err := manifestIDFromValues(pmh.QueryValues)
	if err != nil {
		return err, http.StatusNotFound
	}

	dec := json.NewDecoder(pmh.Request.Body)
	m := &sous.Manifest{}
	dec.Decode(m)
	pmh.State.Manifests.Set(mid, m)
	if err := pmh.StateWriter.WriteState(pmh.State); err != nil {
		return err, http.StatusConflict
	}
	return m, http.StatusOK
}

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
*/

func manifestIDFromValues(qv *QueryValues) (sous.ManifestID, error) {
	var r, o, f string
	var err error
	err = firsterr.Returned(
		func() error { r, err = qv.Single("repo"); return err },
		func() error { o, err = qv.Single("offset", ""); return err },
		func() error { f, err = qv.Single("flavor", ""); return err },
	)
	if err != nil {
		return sous.ManifestID{}, err
	}

	return sous.ManifestID{
		Source: sous.SourceLocation{
			Repo: r,
			Dir:  o,
		},
		Flavor: f,
	}, nil
}
