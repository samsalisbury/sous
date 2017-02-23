package singularity

import (
	"context"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/coaxer"
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
	Cluster sous.Cluster
	promise coaxer.Promise
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

func (rc *requestContext) newDeployStateBuilder() (*DeployStateBuilder, error) {
	return newDeployStateBuilder(rc)
}
