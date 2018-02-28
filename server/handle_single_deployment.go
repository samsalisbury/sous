package server

import (
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
		DeploymentID     sous.DeploymentID
		DeploymentIDErr  error
		Body             *singleDeploymentBody
		BodyErr          error
		Header           http.Header
		GDM              *sous.State
		StateWriter      sous.StateWriter
		GDMToDeployments func(*sous.State) (sous.Deployments, error)
		PushToQueueSet   func(r *sous.Rectification) (*sous.QueuedR11n, bool)
		User             sous.User
	}

	// singleDeploymentBody is the response struct returned from handlers
	// of HTTP methods of a SingleDeploymentResource.
	singleDeploymentBody struct {
		Meta           ResponseMeta
		DeploymentID   sous.DeploymentID
		DeploySpec     sous.DeploySpec
		ManifestHeader sous.Manifest
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

// Put returns a configured put single deployment handler.
func (sdr *SingleDeploymentResource) Put(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	qv := restful.QueryValues{Values: req.URL.Query()}
	did, didErr := deploymentIDFromValues(qv)
	return &PUTSingleDeploymentHandler{
		User:            sous.User(sdr.userExtractor.GetUser(req)),
		DeploymentID:    did,
		DeploymentIDErr: didErr,
	}
}

// Exchange triggers a deployment action when receiving
// a Manifest containing a deployment matching DeploymentID that differs
// from the current actual deployment set. It first writes the new
// deployment spec to the GDM.
func (psd *PUTSingleDeploymentHandler) Exchange() (interface{}, int) {
	er := func(code int, f string, a ...interface{}) (*singleDeploymentBody, int) {
		psd.Body.Meta.Error = fmt.Sprintf(f, a...)
		psd.Body.Meta.StatusCode = code
		return psd.Body, code
	}
	success := func(code int) (interface{}, int) {
		psd.Body.Meta.StatusCode = code
		return psd.Body, code
	}

	did := psd.Body.DeploymentID

	if did != psd.DeploymentID {
		return er(400, "Body contains deployment %q, URL query is for deployment %q.", did, psd.DeploymentID)
	}

	m, ok := psd.GDM.Manifests.Get(did.ManifestID)
	if !ok {
		return er(404, "No manifest with ID %q.", did.ManifestID)
	}

	cluster := psd.Body.DeploymentID.Cluster
	d, ok := m.Deployments[cluster]
	if !ok {
		return er(404, "No %q deployment defined for %q.", cluster, did)
	}
	different, _ := psd.Body.DeploySpec.Diff(d)
	if !different {
		return success(200)
	}

	m.Deployments[cluster] = psd.Body.DeploySpec

	if err := psd.StateWriter.WriteState(psd.GDM, psd.User); err != nil {
		return er(500, "Failed to write state: %s.", err)
	}

	// The full deployment can only be gotten from the full state, since it
	// relies on State.Defs which is not part of this exchange. Therefore
	// fish it out of the realized GDM returned from .Deployments()
	//
	// TODO SS:
	// Note that this call is expensive, we should come up with a cheaper way
	// to get single deployments.
	deployments, err := psd.GDMToDeployments(psd.GDM)
	if err != nil {
		return er(500, "Unable to expand GDM: %s.", err)
	}
	fullDeployment, ok := deployments.Get(psd.DeploymentID)
	if !ok {
		return er(500, "Deployment failed to round-trip to GDM.")
	}

	r := &sous.Rectification{
		Pair: sous.DeployablePair{
			Post: &sous.Deployable{
				Status:     0,
				Deployment: fullDeployment,
			},
			ExecutorData: nil,
		},
	}
	r.Pair.SetID(psd.DeploymentID)

	qr, ok := psd.PushToQueueSet(r)
	if !ok {
		return er(409, "Queue full, please try again later.")
	}

	psd.Body.Meta.Links = map[string]string{
		"queuedDeployAction": "/deploy-queue-item?action=" + string(qr.ID),
	}

	return success(200)
}
