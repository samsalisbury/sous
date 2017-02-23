package singularity

import (
	"context"
	"fmt"
	"log"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/coaxer"
	"github.com/opentable/sous/util/firsterr"
	"github.com/pkg/errors"
)

// requestContext is a mixin used for DeploymentBuilder and DeployStateBuilder.
type requestContext struct {
	//adsBuild adsBuild
	// Client to be used for all requests to Singularity.
	Client   DeployReader
	Registry sous.Registry
	Context  context.Context
	// RequestID is the singularity request ID this builder is working on.
	// This field is populated by NewDeployStateBuilder, and you can always
	// assume that it is populated with a meaningful value.
	RequestID string
	// The cluster which this request belongs to.
	Cluster                  sous.Cluster
	promise                  coaxer.Promise
	DeployHistoryListBuilder *DeployHistoryListBuilder
	// DeployHistoryBuilders is a map of deploy ID to DeployHistoryBuilder.
	DeployHistoryBuilders map[string]*DeployHistoryBuilder
}

// newRequestContext initialises a requestContext and begins making HTTP
// requests to get the request (via coaxer). We can access the results of
// this via the returned requestContext's promise field.
func newRequestContext(ctx context.Context, requestID string, client DeployReader, cluster sous.Cluster, registry sous.Registry) *requestContext {
	rc := &requestContext{
		Client:    client,
		Registry:  registry,
		Context:   ctx,
		Cluster:   cluster,
		RequestID: requestID,
		promise: c.Coax(ctx, func() (interface{}, error) {
			return maybeRetryable(client.GetRequest(requestID))
		}, "get singularity request %q", requestID),
		DeployHistoryBuilders: map[string]*DeployHistoryBuilder{},
	}
	dhb, err := rc.newDeployHistoryBuilder()
	if err != nil {
		// TODO not panic
		panic(err)
	}
	rc.DeployHistoryListBuilder = dhb
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

func (rc *requestContext) CurrentDeployID() (string, error) {
	deployID, _, err := rc.currentDeployIDAndStatus()
	if err != nil {
		return "", err
	}
	return deployID, nil
}

func (rc *requestContext) CurrentDeployStatus() (sous.DeployStatus, error) {
	_, status, err := rc.currentDeployIDAndStatus()
	if err != nil {
		return sous.DeployStatusUnknown, err
	}
	return status, nil
}

// DeployState returns the Sous deploy state for this request.
func (rc *requestContext) DeployState() (*sous.DeployState, error) {
	currentDeployID, currentDeployStatus, err := rc.currentDeployIDAndStatus()
	if err != nil {
		return nil, err
	}
	log.Printf("Gathering deploy state for current deploy %q; request %q", currentDeployID, rc.RequestID)

	// TODO: make newDeploymentBuilder a method on requestContext.
	//currentDeployBuilder := rc.newDeploymentBuilder(currentDeployID)

	// DeployStatusNotRunning means that there is no active or pending deploy.
	if currentDeployStatus == sous.DeployStatusNotRunning {
		// TODO: Check if this should be a retryable error or not?
		//       Maybe there is a race condition where there will be
		//       no active or pending deploy just after a rectify.
		return &sous.DeployState{
			Status: sous.DeployStatusNotRunning,
		}, nil
	}

	// Examine the deploy history to see if the last deployment failed.
	deployHistory, err := rc.DeployHistoryListBuilder.DeployHistoryList()
	if err != nil {
		return nil, err
	}

	if len(deployHistory) == 0 {
		// There has never been a deployment.
		return &sous.DeployState{
			Status: sous.DeployStatusNotRunning,
		}, nil
	}

	lastAttemptedDeployID := deployHistory[0].DeployMarker.DeployId

	var currentDeploy, lastAttemptedDeploy *sous.Deployment
	var lastAttemptedDeployStatus sous.DeployStatus

	if err := firsterr.Set(
		func(err *error) {
			currentDeploy, currentDeployStatus, *err = rc.Deployment(currentDeployID)
		},
		func(err *error) {
			lastAttemptedDeploy, lastAttemptedDeployStatus, *err = rc.Deployment(lastAttemptedDeployID)
		},
	); err != nil {
		return nil, errors.Wrapf(err, "building deploy state")
	}

	return &sous.DeployState{
		Status:     lastAttemptedDeployStatus,
		Deployment: *lastAttemptedDeploy,
	}, nil
}

// Deployment returns the sous.Deployment constructed from the request and
// deploy deployID.
func (rc *requestContext) Deployment(deployID string) (*sous.Deployment, sous.DeployStatus, error) {
	dhb, ok := rc.DeployHistoryBuilders[deployID]
	if !ok {
		dhb = newDeploymentHistoryBuilder(rc, deployID)
	}

	dh, err := dhb.DeployHistory()
	if err != nil {
		return nil, sous.DeployStatusUnknown, err
	}

	deploy := dh.Deploy

	if deploy.ContainerInfo == nil ||
		deploy.ContainerInfo.Docker == nil ||
		deploy.ContainerInfo.Docker.Image == "" {
		return nil, sous.DeployStatusUnknown, fmt.Errorf("no docker image specified at deploy.ContainerInfo.Docker.Image")
	}
	dockerImage := deploy.ContainerInfo.Docker.Image

	// This is our only dependency on the registry.
	labels, err := rc.Registry.ImageLabels(dockerImage)
	sourceID, err := docker.SourceIDFromLabels(labels)
	if err != nil {
		return nil, sous.DeployStatusUnknown, errors.Wrapf(err, "getting source ID")
	}

	requestParent, err := rc.RequestParent()
	if err != nil {
		return nil, sous.DeployStatusUnknown, err
	}
	if requestParent.Request == nil {
		return nil, sous.DeployStatusUnknown, fmt.Errorf("requestParent contains no request")
	}

	return mapDeployHistoryToDeployment(rc.Cluster, sourceID, requestParent.Request, dh)
}

// currentDeployIDAndStatus returns, in order of preference:
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
func (rc *requestContext) currentDeployIDAndStatus() (string, sous.DeployStatus, error) {
	rp, err := rc.RequestParent()
	if err != nil {
		return "", sous.DeployStatusUnknown, err
	}
	rds := rp.RequestDeployState
	if rds == nil {
		return "", sous.DeployStatusUnknown, err
	}

	// If there is a pending request, that's the one we care about from Sous'
	// point of view, so preferentially return that.
	if pending := rds.PendingDeploy; pending != nil {
		return pending.DeployId, sous.DeployStatusPending, nil
	}
	// If there's nothing pending, let's try to return the active deploy.
	if active := rds.ActiveDeploy; active != nil {
		return active.DeployId, sous.DeployStatusActive, nil
	}
	return "", sous.DeployStatusNotRunning, nil
}

func (rc *requestContext) newDeployHistoryBuilder() (*DeployHistoryListBuilder, error) {
	return newDeployHistoryListBuilder(rc)
}

func (rc *requestContext) newDeploymentBuilder(deployID string) *DeployHistoryBuilder {
	return newDeploymentHistoryBuilder(rc, deployID)
}
