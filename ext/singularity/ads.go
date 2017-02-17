package singularity

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/coaxer"
	"github.com/opentable/sous/util/firsterr"
	"github.com/pkg/errors"
)

// c is a temporary global, it will be moved somewhere more sensible soon.
var c = coaxer.NewCoaxer(func(c *coaxer.Coaxer) {
	messages := make(chan string)
	go func() {
		for m := range messages {
			log.Println(m)
		}
	}()
	c.DebugFunc = func(desc string) {
		messages <- desc
	}
})

// Deployer implements sous.Deployer for a single sous Cluster running on
// Singularity.
type Deployer struct {
	Registry sous.Registry
	Client   *singularity.Client
	Cluster  sous.Cluster
}

// adsBuild represents the building of a single sous.DeployStates from a
// single Singularity-hosted cluster.
type adsBuild struct {
	Context       context.Context
	Client        *singularity.Client
	Cluster       sous.Cluster
	Registry      sous.Registry
	ErrorCallback func(error)
}

// requestContext is a mixin used for DeploymentBuilder and DeployStateBuilder.
type requestContext struct {
	adsBuild adsBuild
	// Client to be used for all requests to Singularity.
	Client *singularity.Client
	// RequestID is the singularity request ID this builder is working on.
	// This field is populated by NewDeployStateBuilder, and you can always
	// assume that it is populated with a meaningful value.
	RequestID string
	promise   coaxer.Promise
}

// newRequestContext initialises a requestContext and begins making HTTP
// requests to get the request (via coaxer). We can access the results of
// this via the returned requestContext's promise field.
func (ab *adsBuild) newRequestContext(requestID string) requestContext {
	rc := requestContext{
		adsBuild:  *ab,
		Client:    ab.Client,
		RequestID: requestID,
		promise: c.Coax(ab.Context, func() (interface{}, error) {
			return ab.Client.GetRequest(requestID)
		}, "get singularity request %q", requestID),
	}
	return rc
}

// RequestParent returns the request parent if it was eventually retrieved, or
// an error if the retrieve failed.
func (rc *requestContext) RequestParent() (*dtos.SingularityRequestParent, error) {
	if err := rc.promise.Err(); err != nil {
		return nil, err
	}
	return rc.promise.Value().(*dtos.SingularityRequestParent), nil
}

// DeployStateBuilder gathers information about the state of deployments.
type DeployStateBuilder struct {
	requestContext
	// CurrentDeployID is the singularity deploy ID of the currently active or
	// pending deployment for request RequestID. It may be empty if there is no
	// active or pending deployment.
	CurrentDeployID string
	// CurrentDeployStatus is the status of the current singularity deployment,
	// either sous.DeployStatusActive or sous.DeployStatusComing.
	CurrentDeployStatus sous.DeployStatus
	// PreviousDeployID is the ID of the previous deploy for request
	// requestContext.RequestID.
	PreviousDeployID string
	// PreviousDeployStatus is the status of the previous deployment.
	PreviousDeployStatus sous.DeployStatus
}

// DeploymentBuilder is responsible for constructing a sous.Deployment from a
// Singularity deployment.
type DeploymentBuilder struct {
	requestContext
	promise coaxer.Promise
	// DeployID is the singularity deploy ID
	// (not to be confused with sous.DeployID).
	// You can always expect DeployID to have a meaningful value.
	DeployID string
	Status   sous.DeployStatus
}

func newADSBuild(ctx context.Context, client *singularity.Client, reg sous.Registry, cluster sous.Cluster) *adsBuild {
	return &adsBuild{
		Client:        client,
		Registry:      reg,
		Cluster:       cluster,
		ErrorCallback: func(err error) { log.Println(err) },
		Context:       ctx,
	}
}

// RunningDeployments uses a new adsBuild to construct sous deploy states.
func (d *Deployer) RunningDeployments() (sous.DeployStates, error) {
	return newADSBuild(context.TODO(), d.Client, d.Registry, d.Cluster).DeployStates()
}

// DeployStates returns all deploy states.
func (ab *adsBuild) DeployStates() (sous.DeployStates, error) {

	log.Printf("Getting all requests...")

	// Grab the list of all requests from Singularity.
	requests, err := ab.Client.GetRequests()
	if err != nil {
		return sous.NewDeployStates(), err
	}

	log.Printf("Got: %d requests", len(requests))

	deployStates := sous.NewDeployStates()
	var wg sync.WaitGroup
	errChan := make(chan error)

	// Start gathering all requests concurrently.
gather:
	for _, r := range requests {
		select {
		case <-ab.Context.Done():
			log.Printf("Context ended before all deployments gathered.")
			break gather
		default:
		}

		request := r
		log.Printf("Gathering data for request %q in background.", request.Request.Id)

		wg.Add(1)
		go func() {
			defer wg.Done()
			dsb := ab.newDeployStateBuilder(request)
			ds, err := dsb.DeployState()
			if err != nil {
				ab.ErrorCallback(err)
				errChan <- err
				return
			}
			deployStates.Add(ds)
		}()
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Wait for either error or channel close.
	if err := <-errChan; err != nil {
		return sous.NewDeployStates(), err
	}

	return deployStates, nil
}

func (ab *adsBuild) Errorf(format string, a ...interface{}) error {
	prefix := fmt.Sprintf("reading from cluster %q", ab.Cluster.Name)
	message := fmt.Sprintf(format, a...)
	return fmt.Errorf("%s: %s", prefix, message)
}

// Errorf returns a formatted error with contextual info about which Singularity
// deploy the error relates to.
func (db *DeploymentBuilder) Errorf(format string, a ...interface{}) error {
	prefix := fmt.Sprintf("singularity deployment %q (request %q)", db.DeployID, db.RequestID)
	message := fmt.Sprintf(format, a...)
	return fmt.Errorf("%s: %s", prefix, message)
}

func (ab *adsBuild) newDeployStateBuilder(rp *dtos.SingularityRequestParent) *DeployStateBuilder {
	deployID, status := getCurrentDeployIDAndStatus(rp.RequestDeployState)
	return &DeployStateBuilder{
		requestContext:      ab.newRequestContext(rp.Request.Id),
		CurrentDeployID:     deployID,
		CurrentDeployStatus: status,
	}
}

func getCurrentDeployIDAndStatus(rds *dtos.SingularityRequestDeployState) (string, sous.DeployStatus) {
	// If there is a pending request, that's the one we care about from Sous'
	// point of view, so preferentially return that.
	if pending := rds.PendingDeploy; pending != nil {
		return pending.DeployId, sous.DeployStatusPending
	}
	// If there's nothing pending, let's try to return the active deploy.
	if active := rds.ActiveDeploy; active != nil {
		return active.DeployId, sous.DeployStatusActive
	}
	return "", sous.DeployStatusNotRunning
}

// Deployment returns the Deployment.
func (db *DeploymentBuilder) Deployment() (*sous.Deployment, error) {
	if err := db.promise.Err(); err != nil {
		return nil, err
	}
	deployHistoryItem := db.promise.Value().(*dtos.SingularityDeployHistory)

	deploy := deployHistoryItem.Deploy

	if deploy.ContainerInfo == nil ||
		deploy.ContainerInfo.Docker == nil ||
		deploy.ContainerInfo.Docker.Image == "" {
		return nil, db.Errorf("no docker image specified at deploy.ContainerInfo.Docker.Image")
	}
	dockerImage := deploy.ContainerInfo.Docker.Image

	// This is our only dependency on the registry.
	labels, err := db.adsBuild.Registry.ImageLabels(dockerImage)
	sourceID, err := docker.SourceIDFromLabels(labels)
	if err != nil {
		return nil, errors.Wrapf(err, "getting source ID")
	}

	requestParent, err := db.RequestParent()
	if err != nil {
		return nil, err
	}
	if requestParent.Request == nil {
		return nil, db.Errorf("requestParent contains no request")
	}

	return mapDeployHistoryToDeployment(db.adsBuild.Cluster, sourceID, requestParent.Request, deployHistoryItem)
}

// DeployState returns the Sous deploy state.
func (ds *DeployStateBuilder) DeployState() (*sous.DeployState, error) {

	//var previousDeployID = "TODO: Get previous deployID"

	log.Printf("Gathering deploy state for current deploy %q; request %q", ds.CurrentDeployID, ds.RequestID)
	//log.Printf("Gathering deploy state for previous deploy %q; request %q", previousDeployID, ds.RequestID)

	currentDeployBuilder := ds.newDeploymentBuilder(ds.CurrentDeployID, ds.CurrentDeployStatus)
	//previousDeployBuilder := ds.newDeploymentBuilder(previousDeployID)

	var current *sous.Deployment

	overallStatus := ds.CurrentDeployStatus
	// TODO: If last deploy failed, overall status should reflect that.

	if err := firsterr.Set(
		func(err *error) { current, *err = currentDeployBuilder.Deployment() },
		//func(err *error) { previous, *err = previousDeployBuilder.Deployment() },
	); err != nil {
		return nil, errors.Wrapf(err, "building deploy state")
	}

	return &sous.DeployState{
		Deployment: *current,
		Status:     overallStatus,
	}, nil
}

func (ds *DeployStateBuilder) newDeploymentBuilder(deployID string, status sous.DeployStatus) *DeploymentBuilder {

	promise := c.Coax(ds.adsBuild.Context, func() (interface{}, error) {
		return ds.Client.GetDeploy(ds.RequestID, deployID)
	}, "get deployment %q", deployID)

	return &DeploymentBuilder{
		requestContext: ds.requestContext,
		DeployID:       deployID,
		Status:         status,
		promise:        promise,
	}
}
