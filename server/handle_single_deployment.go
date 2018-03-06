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
	// PUTSingleDeploymentHandler handles manifests containing single deployment
	// specs. See Exchange method for more details.
	PUTSingleDeploymentHandler struct {
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

// Put returns a configured put single deployment handler.
func (sdr *SingleDeploymentResource) Put(rm *restful.RouteMap, rw http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	qv := restful.QueryValues{Values: req.URL.Query()}
	did, didErr := deploymentIDFromValues(qv)
	body := &SingleDeploymentBody{}
	bodyErr := json.NewDecoder(req.Body).Decode(body)
	gdm := sdr.context.liveState()
	return &PUTSingleDeploymentHandler{
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

// err returns the current Body of psd and the provided status code.
// It ensures Meta.StatusCode is also set to the provided code.
// It sets Meta.Error to a formatted error using format f and args a...
func (psd *PUTSingleDeploymentHandler) err(code int, f string, a ...interface{}) (*SingleDeploymentBody, int) {
	psd.Body.Meta.Error = fmt.Sprintf(f, a...)
	psd.Body.Meta.StatusCode = code
	return psd.Body, code
}

// ok returns the current body of psd and the provided status code.
// It ensures Meta.StatusCode is also set to the provided code.
// It sets Meta.Links to the provided links.
func (psd *PUTSingleDeploymentHandler) ok(code int, links map[string]string) (*SingleDeploymentBody, int) {
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
