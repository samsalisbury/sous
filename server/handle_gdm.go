package server

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// GDMResource is the resource for the GDM
	GDMResource struct{}

	// GETGDMHandler is an injectable request handler
	GETGDMHandler struct {
		*sous.LogSet
		GDM *LiveGDM
	}

	// PUTGDMHandler is an injectable request handler
	PUTGDMHandler struct {
		*http.Request
		*sous.LogSet
		GDM          *LiveGDM
		StateManager *graph.StateManager
		StateContext StateContext
	}

	gdmWrapper struct {
		Deployments []*sous.Deployment
	}
)

// Get implements Getable on GDMResource
func (gr *GDMResource) Get() restful.Exchanger { return &GETGDMHandler{} }

// Exchange implements the Handler interface
func (h *GETGDMHandler) Exchange() (interface{}, int) {
	sous.Log.Debug.Print(h.GDM)
	data := gdmWrapper{Deployments: make([]*sous.Deployment, 0)}
	keys := sous.DeploymentIDSlice(h.GDM.Keys())
	sort.Sort(keys)

	for _, k := range keys {
		d, has := h.GDM.Get(k)
		if !has {
			return "Error serializing GDM", http.StatusInternalServerError
		}
		data.Deployments = append(data.Deployments, d)
	}

	return data, http.StatusOK
}

// Put implements Putable on GDMResource
func (gr *GDMResource) Put() restful.Exchanger { return &PUTGDMHandler{} }

// Exchange implements the Handler interface
func (h *PUTGDMHandler) Exchange() (interface{}, int) {
	sous.Log.Debug.Print(h.GDM)

	data := gdmWrapper{}
	dec := json.NewDecoder(h.Request.Body)
	dec.Decode(&data)
	deps := sous.NewDeployments(data.Deployments...)

	state, err := h.StateManager.ReadState(sous.StateContext(h.StateContext))
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

	c := sous.StateContext(h.StateContext)
	if err := h.StateManager.WriteState(state, c); err != nil {
		h.Warn.Printf("%#v", err)
		return "Error committing state", http.StatusInternalServerError
	}

	return "", http.StatusNoContent
}
