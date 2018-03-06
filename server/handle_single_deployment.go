package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// SingleDeploymentResource creates handlers for dealing with single
	// deployments.
	SingleDeploymentResource struct {
		userExtractor
		context ComponentLocator
	}
	// PUTSingleDeploymentHandler updates manifests containing single deployment
	// specs. See Exchange method for more details.
	PUTSingleDeploymentHandler struct {
		SingleDeploymentHandler
		BodyErr          error
		StateWriter      sous.StateWriter
		GDMToDeployments func(*sous.State) (sous.Deployments, error)
		PushToQueueSet   func(r *sous.Rectification) (*sous.QueuedR11n, bool)
		routeMap         *restful.RouteMap
	}

	// GETSingleDeploymentHandler retrieves manifests containing single deployment
	// specs. See Exchange method for more details.
	GETSingleDeploymentHandler struct {
		SingleDeploymentHandler
	}

	// SingleDeploymentHandler contains common data and methods to both
	// the GET and PUT handlers.
	SingleDeploymentHandler struct {
		Body            *SingleDeploymentBody
		DeploymentID    sous.DeploymentID
		DeploymentIDErr error
		GDM             *sous.State
		User            sous.User
		responseWriter  http.ResponseWriter
		// Added from deployment_manager branch XXX dupes
		User              sous.User
		DeploymentID      sous.DeploymentID
		DeploymentIDErr   error
		Body              *SingleDeploymentBody
		BodyErr           error
		DeploymentManager sous.DeploymentManager
		QueueSet          *sous.R11nQueueSet
		responseWriter    http.ResponseWriter
		routeMap          *restful.RouteMap
	}

	// ResponseMeta contains metadata to include in API response bodies.
	ResponseMeta struct {
		// Links is a set of links related to a response body.
		Links map[string]string
		// Error is the error message returned.
		Error string `json:",omitempty"`
		// StatusCode is the HTTP status code of this response.
		StatusCode int
	}
)

func newSingleDeploymentResource(cl ComponentLocator) *SingleDeploymentResource {
	return &SingleDeploymentResource{
		context: cl,
	}
}

func newSingleDeploymentHandler(req *http.Request, rw http.ResponseWriter, body *SingleDeploymentBody, gdm *sous.State, cl ComponentLocator, u sous.User) SingleDeploymentHandler {
	qv := restful.QueryValues{Values: req.URL.Query()}
	did, didErr := deploymentIDFromValues(qv)
	return SingleDeploymentHandler{
		Body:            body,
		User:            u,
		DeploymentID:    did,
		DeploymentIDErr: didErr,
		GDM:             gdm,
		responseWriter:  rw,
// XXX dupes
		User:              sous.User(sdr.userExtractor.GetUser(req)),
		DeploymentID:      did,
		DeploymentIDErr:   didErr,
		Body:              body,
		BodyErr:           bodyErr,
		DeploymentManager: sdr.context.DeploymentManager,
		QueueSet:          sdr.context.QueueSet,
		responseWriter:    rw,
		routeMap:          rm,
	}
}

// Put returns a configured put single deployment handler.
func (sdr *SingleDeploymentResource) Put(rm *restful.RouteMap, rw http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	gdm := sdr.context.liveState()
	body := &SingleDeploymentBody{}
	bodyErr := json.NewDecoder(req.Body).Decode(body)
	sdh := newSingleDeploymentHandler(req, rw, body, gdm, sdr.context, sous.User(sdr.userExtractor.GetUser(req)))
	return &PUTSingleDeploymentHandler{
		SingleDeploymentHandler: sdh,
		BodyErr:                 bodyErr,
		StateWriter:             sdr.context.StateManager,
		PushToQueueSet:          sdr.context.QueueSet.Push,
		GDMToDeployments: func(s *sous.State) (sous.Deployments, error) {
			return gdm.Deployments()
		},
		routeMap: rm,
	}
}

// Get returns a configured get single deployment handler.
func (sdr *SingleDeploymentResource) Get(rm *restful.RouteMap, rw http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	gdm := sdr.context.liveState()
	body := &SingleDeploymentBody{}
	sdh := newSingleDeploymentHandler(req, rw, body, gdm, sdr.context, sous.User(sdr.userExtractor.GetUser(req)))
	return &GETSingleDeploymentHandler{
		SingleDeploymentHandler: sdh,
	}
}

// Exchange returns a single deployment.
func (h *GETSingleDeploymentHandler) Exchange() (interface{}, int) {

	did := h.DeploymentID

	m, ok := h.GDM.Manifests.Get(did.ManifestID)
	if !ok {
		return h.err(404, "No manifest with ID %q.", did.ManifestID)
	}

	cluster := did.Cluster
	d, ok := m.Deployments[cluster]
	if !ok {
		return h.err(404, "No %q deployment defined for %q.", cluster, did)
	}

	m.Deployments = nil
	h.Body.DeploySpec = d
	h.Body.ManifestHeader = *m

	return h.ok(200, nil)
}

// err returns the current Body of psd and the provided status code.
// It ensures Meta.StatusCode is also set to the provided code.
// It sets Meta.Error to a formatted error using format f and args a...
func (psd *SingleDeploymentHandler) err(code int, f string, a ...interface{}) (*SingleDeploymentBody, int) {
	psd.Body.Meta.Error = fmt.Sprintf(f, a...)
	psd.Body.Meta.StatusCode = code
	return psd.Body, code
}

// ok returns the current body of psd and the provided status code.
// It ensures Meta.StatusCode is also set to the provided code.
// It sets Meta.Links to the provided links.
func (psd *SingleDeploymentHandler) ok(code int, links map[string]string) (*SingleDeploymentBody, int) {
	psd.Body.Meta.StatusCode = code
	psd.Body.Meta.Links = links
	return psd.Body, code
}

// Exchange triggers a deployment action when receiving
// a Manifest containing a deployment matching DeploymentID that differs
// from the current actual deployment set. It first writes the new
// deployment spec to the GDM.
func (psd *PUTSingleDeploymentHandler) Exchange() (interface{}, int) {

	if psd.BodyErr != nil {
		psd.Body = &SingleDeploymentBody{}
		return psd.err(400, "Error parsing body: %s.", psd.BodyErr)
	}

	did := psd.DeploymentID

	dep, err := psd.DeploymentManager.ReadDeployment(did)
	if err != nil {
		return psd.err(404, "No manifest with ID %q. %v", did.ManifestID, err)
	}

	different, _ := psd.Body.DeploySpec.Diff(dep.DeploySpec())
	if !different {
		return psd.ok(200, nil)
	}

	dep.SourceID.Version = psd.Body.DeploySpec.Version
	dep.DeployConfig = psd.Body.DeploySpec.DeployConfig

	if err := psd.DeploymentManager.WriteDeployment(dep, psd.User); err != nil {
		return psd.err(500, "Failed to write deployment: %s.", err)
	}

	r := &sous.Rectification{
		Pair: sous.DeployablePair{
			Post: &sous.Deployable{
				Status:     0,
				Deployment: dep,
			},
			ExecutorData: nil,
		},
	}
	r.Pair.SetID(psd.DeploymentID)

	qr, ok := psd.QueueSet.Push(r)
	if !ok {
		return psd.err(409, "Queue full, please try again later.")
	}

	actionKV := restful.KV{"action", string(qr.ID)}
	queueURI, err := psd.routeMap.URIFor("deploy-queue-item", nil, actionKV)
	if err == nil {
		psd.responseWriter.Header().Add("Location", queueURI)
	}

	return psd.ok(201, map[string]string{
		"queuedDeployAction": "/deploy-queue-item?action=" + string(qr.ID),
	})
}
