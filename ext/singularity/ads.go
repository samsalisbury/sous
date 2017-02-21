package singularity

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

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
	c.Backoff = time.Second
})

// DeployReader encapsulates the methods required to read Singularity
// requests and deployments.
type DeployReader interface {
	GetRequests() (dtos.SingularityRequestParentList, error)
	GetRequest(requestID string) (*dtos.SingularityRequestParent, error)
	GetDeploy(requestID, deployID string) (*dtos.SingularityDeployHistory, error)
}

// Deployer implements sous.Deployer for a single sous Cluster running on
// Singularity.
type Deployer struct {
	Registry      sous.Registry
	ClientFactory func(*sous.Cluster) DeployReader
	Clusters      sous.Clusters
}

// adsBuild represents the building of a single sous.DeployStates from a
// single Singularity-hosted cluster.
type adsBuild struct {
	Context       context.Context
	ClientFactory func(*sous.Cluster) DeployReader
	Clusters      sous.Clusters
	Registry      sous.Registry
	ErrorCallback func(error)
}

// requestContext is a mixin used for DeploymentBuilder and DeployStateBuilder.
type requestContext struct {
	adsBuild adsBuild
	// Client to be used for all requests to Singularity.
	Client DeployReader
	// RequestID is the singularity request ID this builder is working on.
	// This field is populated by NewDeployStateBuilder, and you can always
	// assume that it is populated with a meaningful value.
	RequestID string
	// The cluster which this request belongs to.
	Cluster sous.Cluster
	promise coaxer.Promise
}

// newRequestContext initialises a requestContext and begins making HTTP
// requests to get the request (via coaxer). We can access the results of
// this via the returned requestContext's promise field.
func (ab *adsBuild) newRequestContext(requestID string, client DeployReader, cluster sous.Cluster) requestContext {
	rc := requestContext{
		adsBuild:  *ab,
		Client:    client,
		Cluster:   cluster,
		RequestID: requestID,
		promise: c.Coax(ab.Context, func() (interface{}, error) {
			return maybeRetryable(client.GetRequest(requestID))
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

func newADSBuild(ctx context.Context, client func(*sous.Cluster) DeployReader, reg sous.Registry, clusters sous.Clusters) *adsBuild {
	return &adsBuild{
		ClientFactory: client,
		Registry:      reg,
		Clusters:      clusters,
		ErrorCallback: func(err error) { log.Println(err) },
		Context:       ctx,
	}
}

// RunningDeployments uses a new adsBuild to construct sous deploy states.
func (d *Deployer) RunningDeployments() (sous.DeployStates, error) {
	return newADSBuild(context.TODO(), d.ClientFactory, d.Registry, d.Clusters).DeployStates()
}

// DeployStates returns all deploy states.
func (ab *adsBuild) DeployStates() (sous.DeployStates, error) {

	log.Printf("Getting all requests...")

	promises := make(map[string]coaxer.Promise, len(ab.Clusters))

	var requests []*dtos.SingularityRequestParent

	// Grab the list of all requests from all clusters.
	for clusterName, cluster := range ab.Clusters {
		cluster := cluster
		// TODO: Make sous.Clusters a slice to avoid this double-entry record keeping.
		cluster.Name = clusterName
		promises[cluster.Name] = c.Coax(context.TODO(), func() (interface{}, error) {
			if ab.ClientFactory == nil {
				panic("CF")
			}
			if ab.Clusters == nil {
				panic("CLUSTERS")
			}
			if cluster == nil {
				panic("CLUSTER")
			}
			client := ab.ClientFactory(ab.Clusters[cluster.Name])
			return maybeRetryable(client.GetRequests())
		}, "get requests from cluster %q", cluster.Name)
	}

	for cluster, promise := range promises {
		if err := promise.Err(); err != nil {
			log.Printf("Fatal: unable to get requests for cluster %q", cluster)
			return sous.NewDeployStates(), err
		}
		log.Printf("Success: got all requests from cluster %q", cluster)
		requests = append(requests, promise.Value().(dtos.SingularityRequestParentList)...)
	}

	log.Printf("Got: %d requests", len(requests))

	deployStates := sous.NewDeployStates()
	var wg sync.WaitGroup
	errChan := make(chan error)

	// Start gathering all requests concurrently.
gather:
	for _, request := range requests {
		request := request
		select {
		case <-ab.Context.Done():
			log.Printf("Context ended before all deployments gathered.")
			break gather
		default:
		}

		requestID := request.Request.Id

		log.Printf("Gathering data for request %q in background.", requestID)
		deployID, err := ParseRequestID(requestID)
		if err != nil {
			// TODO: Maybe log this?
			continue
		}
		oneOfMyDeploys := false
		for clusterName := range ab.Clusters {
			if deployID.Cluster == clusterName {
				oneOfMyDeploys = true
				break
			}
		}
		if !oneOfMyDeploys {
			// TODO: Maybe log this?
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			dsb := ab.newDeployStateBuilder(deployID.Cluster, request)
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
	//prefix := fmt.Sprintf("reading from cluster %q", ab.Cluster.Name)
	prefix := ""
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

func (ab *adsBuild) newDeployStateBuilder(clusterName string, rp *dtos.SingularityRequestParent) *DeployStateBuilder {
	deployID, status := getCurrentDeployIDAndStatus(rp.RequestDeployState)
	cluster := ab.Clusters[clusterName]
	client := ab.ClientFactory(cluster)
	return &DeployStateBuilder{
		requestContext:      ab.newRequestContext(rp.Request.Id, client, *cluster),
		CurrentDeployID:     deployID,
		CurrentDeployStatus: status,
	}
}

// getCurrentDeployIDAndStatus returns, in order of preference:
//
//     1. Any deploy in PENDING state, as this should be the newest one.
//     2. If there is no PENDING deploy, but there is an ACTIVE deploy, return that.
//     3. If there is no PENDING or ACTIVE deploy, return an empty DeployID and
//        DeployStatusNotRunning so we know there are no deployments here yet.
//
// DeployStatusNotRunning means either there are no deployments here yet, or the
// request is paused, finished, deleted, or in system cooldown. The parent of
// the request deploy state (the RequestParent) has a field "state" that has
// this info if we need it at some point.
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

	return mapDeployHistoryToDeployment(db.Cluster, sourceID, requestParent.Request, deployHistoryItem)
}

// DeployState returns the Sous deploy state.
func (ds *DeployStateBuilder) DeployState() (*sous.DeployState, error) {

	// DeployStatusNotRunning means that there is no active or pending deploy.
	if ds.CurrentDeployStatus == sous.DeployStatusNotRunning {
		// TODO: Check if this should be a retryable error or not?
		//       Maybe there is a race condition where there will be
		//       no active or pending deploy just after a rectify.
		return &sous.DeployState{
			Status: ds.CurrentDeployStatus,
		}, nil
	}

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

type temporary struct {
	error
}

func (t temporary) Temporary() bool {
	return true
}

func maybeRetryable(a interface{}, err error) (interface{}, error) {
	if err == nil {
		return a, nil
	}
	log.Printf("Maybe retryable %T? %q", err, err)
	return a, temporary{err}
}

func (ds *DeployStateBuilder) newDeploymentBuilder(deployID string, status sous.DeployStatus) *DeploymentBuilder {

	promise := c.Coax(ds.adsBuild.Context, func() (interface{}, error) {
		return maybeRetryable(ds.Client.GetDeploy(ds.RequestID, deployID))
	}, "get deployment %q", deployID)

	return &DeploymentBuilder{
		requestContext: ds.requestContext,
		DeployID:       deployID,
		Status:         status,
		promise:        promise,
	}
}
