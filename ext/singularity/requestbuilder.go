package singularity

import (
	"context"
	"fmt"
	"log"

	"github.com/opentable/go-singularity/dtos"
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
	Cluster              sous.Cluster
	promise              coaxer.Promise
	DeployHistoryBuilder *DeployHistoryListBuilder
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
	}
	dhb, err := rc.newDeployHistoryBuilder()
	if err != nil {
		// TODO not panic
		panic(err)
	}
	rc.DeployHistoryBuilder = dhb
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
	currentDeployBuilder := rc.newDeploymentBuilder(currentDeployID)

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
	deployHistory, err := rc.DeployHistoryBuilder.DeployHistory()
	if err != nil {
		return nil, err
	}

	if len(deployHistory) == 0 {
		// There has never been a deployment.
		return &sous.DeployState{
			Status: sous.DeployStatusNotRunning,
		}, nil
	}

	lastDeployHistory := deployHistory[0]
	if lastDeployHistory.Deploy == nil {
		return nil, fmt.Errorf("deploy history item has a nil deploy")
	}

	lastDeployID := deployHistory[0].Deploy.Id
	var lastDeployBuilder *DeploymentBuilder
	// Get the entire last deployment unless it has the same ID as the current
	// one, in which case return the current deployment for the last deployment
	// as well.
	if lastDeployID != currentDeployID {
		log.Printf("Gathering deploy state for last attempted deploy %q; request %q", lastDeployID, rc.RequestID)
		lastDeployBuilder = rc.newDeploymentBuilder(lastDeployID)
	} else {
		lastDeployBuilder = currentDeployBuilder
	}

	var currentDeploy, lastAttemptedDeploy *sous.Deployment
	var lastAttemptedDeployStatus sous.DeployStatus

	if err := firsterr.Set(
		func(err *error) {
			currentDeploy, currentDeployStatus, *err = currentDeployBuilder.Deployment()
		},
		func(err *error) {
			lastAttemptedDeploy, lastAttemptedDeployStatus, *err = lastDeployBuilder.Deployment()
		},
	); err != nil {
		return nil, errors.Wrapf(err, "building deploy state")
	}

	return &sous.DeployState{
		Status:     lastAttemptedDeployStatus,
		Deployment: *lastAttemptedDeploy,
	}, nil
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
	return newDeployHistoryBuilder(rc)
}

func (rc *requestContext) newDeploymentBuilder(deployID string) *DeploymentBuilder {
	return newDeploymentBuilder(rc, deployID)
}
