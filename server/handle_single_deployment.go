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
		DeploymentID     sous.DeploymentID
		DeploymentIDErr  error
		Body             *singleDeploymentBody
		BodyErr          error
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

func newSingleDeploymentResource(cl ComponentLocator) *SingleDeploymentResource {
	return &SingleDeploymentResource{
		context: cl,
	}
}

// Put returns a configured put single deployment handler.
func (sdr *SingleDeploymentResource) Put(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	qv := restful.QueryValues{Values: req.URL.Query()}
	did, didErr := deploymentIDFromValues(qv)
	body := &singleDeploymentBody{}
	bodyErr := json.NewDecoder(req.Body).Decode(body)
	gdm := sdr.context.liveState()
	return &PUTSingleDeploymentHandler{
		User:            sous.User(sdr.userExtractor.GetUser(req)),
		DeploymentID:    did,
		DeploymentIDErr: didErr,
		Body:            body,
		BodyErr:         bodyErr,
		StateWriter:     sdr.context.StateManager,
		PushToQueueSet:  sdr.context.QueueSet.Push,
		GDM:             gdm,
		GDMToDeployments: func(s *sous.State) (sous.Deployments, error) {
			return gdm.Deployments()
		},
	}
}

// err returns the current Body of psd and the provided status code.
// It ensures Meta.StatusCode is also set to the provided code.
// It sets Meta.Error to a formatted error using format f and args a...
func (psd *PUTSingleDeploymentHandler) err(code int, f string, a ...interface{}) (*singleDeploymentBody, int) {
	psd.Body.Meta.Error = fmt.Sprintf(f, a...)
	psd.Body.Meta.StatusCode = code
	return psd.Body, code
}

// ok returns the current body of psd and the provided status code.
// It ensures Meta.StatusCode is also set to the provided code.
// It sets Meta.Links to the provided links.
func (psd *PUTSingleDeploymentHandler) ok(code int, links map[string]string) (*singleDeploymentBody, int) {
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
		psd.Body = &singleDeploymentBody{}
		return psd.err(400, "Error parsing body: %s.", psd.BodyErr)
	}

	did := psd.DeploymentID

	m, ok := psd.GDM.Manifests.Get(did.ManifestID)
	if !ok {
		return psd.err(404, "No manifest with ID %q.", did.ManifestID)
	}

	cluster := did.Cluster
	d, ok := m.Deployments[cluster]
	if !ok {
		return psd.err(404, "No %q deployment defined for %q.", cluster, did)
	}
	different, _ := psd.Body.DeploySpec.Diff(d)
	if !different {
		return psd.ok(200, nil)
	}

	m.Deployments[cluster] = psd.Body.DeploySpec

	if err := psd.StateWriter.WriteState(psd.GDM, psd.User); err != nil {
		return psd.err(500, "Failed to write state: %s.", err)
	}

	// The full deployment can only be gotten from the full state, since it
	// relies on State.Defs which is not part of this exchange. Therefore
	// fish it out of the realized GDM returned from GDMToDeployments.
	//
	// TODO SS:
	// Note that this call is expensive, we should come up with a cheaper way
	// to get single deployments.
	deployments, err := psd.GDMToDeployments(psd.GDM)
	if err != nil {
		return psd.err(500, "Unable to expand GDM: %s.", err)
	}
	fullDeployment, ok := deployments.Get(psd.DeploymentID)
	if !ok {
		return psd.err(500, "Deployment failed to round-trip to GDM.")
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
		return psd.err(409, "Queue full, please try again later.")
	}

	return psd.ok(201, map[string]string{
		"queuedDeployAction": "/deploy-queue-item?action=" + string(qr.ID),
	})
}
