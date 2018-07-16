package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/ext/singularity"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
)

// https://github.com/opentable/sous/blob/0a96ed483cd86abc9604993120e8dd211cf7adc6/server/handle_single_deployment.go

type (
	// SingleDeploymentResource creates handlers for dealing with single
	// deployments.
	SingleDeploymentResource struct {
		context ComponentLocator
	}
	// PUTSingleDeploymentHandler updates manifests containing single deployment
	// specs. See Exchange method for more details.
	PUTSingleDeploymentHandler struct {
		SingleDeploymentHandler
		QueueSet    sous.QueueSet
		routeMap    *restful.RouteMap
		StateWriter sous.StateWriter
	}

	// GETSingleDeploymentHandler retrieves manifests containing single deployment
	// specs. See Exchange method for more details.
	GETSingleDeploymentHandler struct {
		SingleDeploymentHandler
	}

	// SingleDeploymentHandler contains common data and methods to both
	// the GET and PUT handlers.
	SingleDeploymentHandler struct {
		userExtractor
		Body           SingleDeploymentBody
		req            *http.Request
		responseWriter http.ResponseWriter
		GDM            *sous.State
		log            logging.LogSink
	}
)

func newSingleDeploymentResource(cl ComponentLocator) *SingleDeploymentResource {
	return &SingleDeploymentResource{
		context: cl,
	}
}

func (sdr *SingleDeploymentResource) newSingleDeploymentHandler(ls logging.LogSink, req *http.Request, rw http.ResponseWriter, gdm *sous.State) SingleDeploymentHandler {
	return SingleDeploymentHandler{
		responseWriter: rw,
		req:            req,
		GDM:            gdm,
		log:            ls,
	}
}

func (sdh *SingleDeploymentHandler) force() (force bool, err error) {
	qv := restful.QueryValues{Values: sdh.req.URL.Query()}
	return forceFromValues(qv)
}

func (sdh *SingleDeploymentHandler) depID() (sous.DeploymentID, error) {
	qv := restful.QueryValues{Values: sdh.req.URL.Query()}
	return deploymentIDFromValues(qv)
}

// Put returns a configured put single deployment handler.
func (sdr *SingleDeploymentResource) Put(rm *restful.RouteMap, ls logging.LogSink, rw http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	gdm := sdr.context.liveState()
	sdh := sdr.newSingleDeploymentHandler(ls, req, rw, gdm)
	return &PUTSingleDeploymentHandler{
		SingleDeploymentHandler: sdh,
		QueueSet:                sdr.context.QueueSet,
		routeMap:                rm,
		StateWriter:             sdr.context.StateManager,
	}
}

// Get returns a configured get single deployment handler.
func (sdr *SingleDeploymentResource) Get(rm *restful.RouteMap, ls logging.LogSink, rw http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	gdm := sdr.context.liveState()
	sdh := sdr.newSingleDeploymentHandler(ls, req, rw, gdm)
	return &GETSingleDeploymentHandler{SingleDeploymentHandler: sdh}
}

// Exchange returns a single deployment.
func (h *GETSingleDeploymentHandler) Exchange() (interface{}, int) {
	did, err := h.depID()
	if err != nil {
		return h.err(400, "Cannot decode Deployment ID: %s.", err)
	}

	m, ok := h.GDM.Manifests.Get(did.ManifestID)
	if !ok {
		return h.err(404, "No manifest with ID %q", did.ManifestID)
	}

	dep, ok := m.Deployments[did.Cluster]
	if !ok {
		return h.err(404, "Manifest %q has no deployment for cluster %q.", m.ID(), did.Cluster)
	}

	if dep.NumInstances == 0 {
		return h.err(400, "Cannot deploy, current num instances set to zero, please update manifest.")
	}

	links := make(map[string]string)
	baseURL := h.GDM.Defs.Clusters[did.Cluster].BaseURL
	singularityRequestID := dep.DeployConfig.SingularityRequestID
	singularityURL := makeSingularityURL(baseURL, singularityRequestID, did)
	if len(singularityURL) > 0 {
		links["SingularityURL"] = singularityURL
	}

	h.Body.Deployment = &dep

	return h.ok(200, links)
}

func makeSingularityURL(baseURL string, singularityRequestID string, deploymentID sous.DeploymentID) string {
	var err error
	if len(singularityRequestID) == 0 {
		singularityRequestID, err = singularity.MakeRequestID(deploymentID)
		if err != nil {
			return fmt.Sprintf("Unable to determine SingularityRequest URL : %s", err)
		}
	}
	singularityURL, err := singularity.MakeRequestURL(baseURL, singularityRequestID)
	if err != nil {
		singularityURL = fmt.Sprintf("Unable to determine SingularityRequest URL : %s", err)
	}
	return singularityURL
}

// err returns the current Body of psd and the provided status code.
// It ensures Meta.StatusCode is also set to the provided code.
// It sets Meta.Error to a formatted error using format f and args a...
func (sdh *SingleDeploymentHandler) err(code int, f string, a ...interface{}) (interface{}, int) {
	return fmt.Sprintf(f, a...), code
}

// ok returns the current body of psd and the provided status code.
// It ensures Meta.StatusCode is also set to the provided code.
// It sets Meta.Links to the provided links.
func (sdh *SingleDeploymentHandler) ok(code int, links map[string]string) (SingleDeploymentBody, int) {
	sdh.Body.Meta.Links = links
	return sdh.Body, code
}

// Exchange triggers a deployment action when receiving
// a Manifest containing a deployment matching DeploymentID that differs
// from the current actual deployment set. It first writes the new
// deployment spec to the GDM.
func (psd *PUTSingleDeploymentHandler) Exchange() (interface{}, int) {
	did, err := psd.depID()
	if err != nil {
		return psd.err(400, "Cannot decode Deployment ID: %s.", err)
	}

	force, err := psd.force()
	if err != nil {
		return psd.err(400, "Cannot parse force from client: %s", err)
	}

	if err := json.NewDecoder(psd.req.Body).Decode(&psd.Body); err != nil {
		return psd.err(400, "Error parsing body: %s.", err)
	}

	if psd.Body.Deployment == nil {
		return psd.err(400, "Body.Deployment is nil.")
	}

	messages.ReportLogFieldsMessageToConsole("Exchange PutSingleDeplymentHandler", logging.ExtraDebug1Level, psd.log, did, psd.Body)

	if _, clusterOK := psd.GDM.Defs.Clusters[did.Cluster]; !clusterOK {
		return psd.err(404, "Cluster %q not defined.", did.Cluster)
	}

	m, ok := psd.GDM.Manifests.Get(did.ManifestID)
	if !ok {
		return psd.err(404, "No manifest with ID %q.", did.ManifestID)
	}
	original, ok := m.Deployments[did.Cluster]
	if !ok {
		return psd.err(404, "Manifest %q has no deployment for cluster %q.",
			did.ManifestID, did.Cluster)
	}

	if psd.Body.Deployment.NumInstances == 0 {
		return psd.err(400, "Cannot deploy: NumInstances is 0 for this deployment. Please update your manifest to NumInstances > 0 to enable deploying.")
	}

	different, _ := psd.Body.Deployment.Diff(original)
	if !different && !force {
		return psd.ok(200, nil)
	}

	m.Deployments[did.Cluster] = *psd.Body.Deployment

	user := sous.User(psd.GetUser(psd.req))

	if err := psd.StateWriter.WriteState(psd.GDM, user); err != nil {
		return psd.err(500, "Failed to write state: %s.", err)
	}

	// Round-trip the updated GDM back to deployments to check validity.
	deployments, err := psd.GDM.Deployments()
	if err != nil {
		return psd.err(500, "Failed to round-trip new deployment spec to GDM: %s", err)
	}
	newDeployment, ok := deployments.Get(did)
	if !ok {
		return psd.err(500, "Failed to round-trip new deployment spec to GDM.")
	}

	if flaws := newDeployment.Validate(); len(flaws) != 0 {
		return psd.err(400, "Deployment invalid after round-trip to GDM: %v", flaws)
	}

	newDeployment.User = user

	r := sous.NewRectification(sous.DeployablePair{Post: &sous.Deployable{
		Deployment: newDeployment,
	}}, psd.log.Child("r11n"))

	r.Pair.SetID(did)

	postID := ""
	version := ""
	if r.Pair.Post != nil {
		postID = r.Pair.Post.ID().String()
		version = r.Pair.Post.DeploySpec().Version.String()
	}

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Pushing following onto queue %s:%s", postID, version), logging.ExtraDebug1Level, psd.log, r)

	qr, ok := psd.QueueSet.Push(r)
	if !ok {
		return psd.err(409, "Queue full, please try again later.")
	}

	actionKV := restful.KV{"action", string(qr.ID)}
	clusterKV := restful.KV{"cluster", did.Cluster}
	repoKV := restful.KV{"repo", did.ManifestID.Source.Repo}
	offsetKV := restful.KV{"offset", did.ManifestID.Source.Dir}
	flavorKV := restful.KV{"flavor", did.ManifestID.Flavor}
	hostName := psd.req.Host
	queueURI, err := psd.routeMap.FullURIFor(hostName, "deploy-queue-item", nil,
		actionKV, clusterKV, repoKV, offsetKV, flavorKV)

	if err != nil {
		return psd.err(500, "Determining queue item URL: %s", err)
	}
	if err == nil {
		psd.responseWriter.Header().Add("Location", queueURI)
	}

	return psd.ok(201, map[string]string{"queuedDeployAction": queueURI})
}
