package server

import (
	"encoding/json"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
)

type (
	// ManifestResource describes resources for manifests
	ManifestResource struct {
		userExtractor
		restful.QueryParser
		context ComponentLocator
	}

	// GETManifestHandler handles GET exchanges for manifests
	GETManifestHandler struct {
		*sous.State
		restful.QueryValues
	}

	// PUTManifestHandler handles PUT exchanges for manifests
	PUTManifestHandler struct {
		*sous.State
		*logging.LogSet
		*http.Request
		restful.QueryValues
		User        ClientUser
		StateWriter sous.StateWriter
	}

	// DELETEManifestHandler handles DELETE exchanges for manifests
	DELETEManifestHandler struct {
		*sous.State
		restful.QueryValues
		StateWriter sous.StateWriter
	}
)

func newManifestResource(ctx ComponentLocator) *ManifestResource {
	return &ManifestResource{context: ctx}
}

// Get implements Getable for ManifestResource
func (mr *ManifestResource) Get(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &GETManifestHandler{
		State:       mr.context.liveState(),
		QueryValues: mr.ParseQuery(req),
	}
}

// Put implements Putable for ManifestResource
func (mr *ManifestResource) Put(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &PUTManifestHandler{
		State:       mr.context.liveState(),
		LogSet:      mr.context.LogSet,
		Request:     req,
		QueryValues: mr.ParseQuery(req),
		User:        mr.GetUser(req),
		StateWriter: sous.StateWriter(mr.context.StateManager),
	}
}

// Delete implements Deleteable for ManifestResource
func (mr *ManifestResource) Delete(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &DELETEManifestHandler{
		State:       mr.context.liveState(),
		QueryValues: mr.ParseQuery(req),
		StateWriter: sous.StateWriter(mr.context.StateManager),
	}
}

// Exchange implements restful.Exchanger
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

// Exchange implements restful.Exchanger
func (dmh *DELETEManifestHandler) Exchange() (interface{}, int) {
	mid, err := manifestIDFromValues(dmh.QueryValues)
	if err != nil {
		return err, http.StatusNotFound
	}
	_, there := dmh.State.Manifests.Get(mid)
	if !there {
		return nil, http.StatusNotFound
	}
	dmh.State.Manifests.Remove(mid)

	return nil, http.StatusNoContent
}

// Exchange implements restful.Exchanger
func (pmh *PUTManifestHandler) Exchange() (interface{}, int) {
	mid, err := manifestIDFromValues(pmh.QueryValues)
	if err != nil {
		return err, http.StatusNotFound
	}

	m := &sous.Manifest{}
	dec := json.NewDecoder(pmh.Request.Body)
	dec.Decode(m)

	flaws := m.Validate()
	if len(flaws) > 0 {
		pmh.Vomit.Print(spew.Sdump(flaws))
		return "Invalid manifest", http.StatusBadRequest
	}
	pmh.State.Manifests.Set(mid, m)
	if err := pmh.StateWriter.WriteState(pmh.State, sous.User(pmh.User)); err != nil {
		return errors.Wrapf(err, "state recording collision - retry"), http.StatusConflict
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

func manifestIDFromValues(qv restful.QueryValues) (sous.ManifestID, error) {
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
