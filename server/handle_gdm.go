package server

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// GDMResource is the resource for the GDM
	GDMResource struct {
		userExtractor
		context ComponentLocator
	}

	// GETGDMHandler is an injectable request handler
	GETGDMHandler struct {
		*logging.LogSet
		GDM      *sous.State
		RzWriter http.ResponseWriter
	}

	// PUTGDMHandler is an injectable request handler
	PUTGDMHandler struct {
		*http.Request
		*logging.LogSet
		GDM          *sous.State
		StateManager sous.StateManager
		User         ClientUser
	}
)

func newGDMResource(ctx ComponentLocator) *GDMResource {
	return &GDMResource{context: ctx}
}

// Get implements Getable on GDMResource
func (gr *GDMResource) Get(writer http.ResponseWriter, _ *http.Request, _ httprouter.Params) restful.Exchanger {
	return &GETGDMHandler{
		LogSet:   gr.context.LogSet,
		GDM:      gr.context.LiveState(),
		RzWriter: writer,
	}
}

// Exchange implements the Handler interface
func (h *GETGDMHandler) Exchange() (interface{}, int) {
	logging.Log.Debugf("%v", h.GDM)
	data := GDMWrapper{Deployments: make([]*sous.Deployment, 0)}
	deps, err := h.GDM.Deployments()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	keys := sous.DeploymentIDSlice(deps.Keys())
	sort.Sort(keys)

	for _, k := range keys {
		d, has := deps.Get(k)
		if !has {
			return "Error serializing GDM", http.StatusInternalServerError
		}
		data.Deployments = append(data.Deployments, d)
	}
	etag, _ := h.GDM.GetEtag()
	h.RzWriter.Header().Set("Etag", etag)

	return data, http.StatusOK
}

// Put implements Putable on GDMResource
func (gr *GDMResource) Put(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &PUTGDMHandler{
		Request:      req,
		LogSet:       gr.context.LogSet,
		GDM:          gr.context.LiveState(),
		StateManager: gr.context.StateManager,
		User:         gr.GetUser(req),
	}
}

// Exchange implements the Handler interface
func (h *PUTGDMHandler) Exchange() (interface{}, int) {
	logging.Log.Debug.Print(h.GDM)

	data := GDMWrapper{}
	dec := json.NewDecoder(h.Request.Body)
	dec.Decode(&data)
	deps := sous.NewDeployments(data.Deployments...)

	state, err := h.StateManager.ReadState()
	if err != nil {
		h.Warn.Printf("%#v", err)
		return "Error loading state from storage", http.StatusInternalServerError
	}

	state.Manifests, err = deps.PutbackManifests(state.Defs, state.Manifests)
	if err != nil {
		h.Warn.Printf("%#v", err)
		return "Error getting state", http.StatusConflict
	}

	flaws := state.Validate()
	if len(flaws) > 0 {
		h.Warn.Printf("%#v", flaws)
		return "Invalid GDM", http.StatusBadRequest
	}

	if _, got := h.Header["Etag"]; got {
		state.SetEtag(h.Header.Get("Etag"))
	}

	if err := h.StateManager.WriteState(state, sous.User(h.User)); err != nil {
		h.Warn.Printf("%#v", err)
		return "Error committing state", http.StatusInternalServerError
	}

	return "", http.StatusNoContent
}
