package server

import (
	"encoding/json"
	"net/http"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
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
		StateWriter graph.LocalStateWriter
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
*/

func manifestIDFromValues(qv *QueryValues) (ManifestID, error) {
	var r, o, f string
	repos := qv.Get("repo")
	switch len(repos) {
	case 0:
		return ManifestID{}, errors.New("No repo given")
	case 1:
		r = repos[0]
	default:
		return ManifestID{}, errors.New("Multiple repo given")
	}

	offsets := qv.Get("offset")
	switch len(offsets) {
	case 0:
		return ManifestID{}, errors.New("No offset given")
	case 1:
		o = offsets[0]
	default:
		return ManifestID{}, errors.New("Multiple offsets given")
	}

	flavors := qv.Get("flavor")
	switch len(flavors) {
	case 0:
		return ManifestID{}, errors.New("No flavor given")
	case 1:
		o = flavors[0]
	default:
		return ManifestID{}, errors.New("Multiple flavors given")
	}

	return ManifestID{
		SourceLocation: SourceLocation{
			Repo:   r,
			Offset: o,
		},
		Flavor: f,
	}, nil
}

// Exchange implements Exchanger
func (gmh *GETManifestHandler) Exchange() (interface{}, int) {
	mid, err := manifestIDFromValues(gmh.QueryValues)
	if err != nil {
		return nil, http.StatusGone // Gone because we know this URL will always be wrong
	}
	m, err := gmh.State.Manifests.Get(mid)
	if err != nil {
		return nil, http.StatusGone
	}
	return m, http.StatusOK
}

// Exchange implements Exchanger
func (pmh *PUTManifestHandler) Exchange() (interface{}, int) {
	mid, err := manifestIDFromValues(pmh.QueryValues)
	if err != nil {
		return nil, http.StatusGone
	}

	dec := json.NewDecoder(pmh.Request.Body)
	m := &sous.Manifest{}
	dec.Decode(m)
	pmh.State.Manifests.Set(mid, m)
	if err := pmh.StateWriter.WriteState(pmh.State){
		return nil, http.StatusConflict
	}
	return m, http.StatusOK

	// XXX SAVE the new state, possibly with 409 if push fails
}
