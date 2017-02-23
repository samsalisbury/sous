package singularity

import (
	"context"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/coaxer"
)

// DeployHistoryListBuilder gathers information about the state of deployments.
type DeployHistoryListBuilder struct {
	requestContext *requestContext
	// PreviousDeployID is the ID of the previous deploy for request
	// requestContext.RequestID.
	PreviousDeployID string
	// PreviousDeployStatus is the status of the previous deployment.
	PreviousDeployStatus sous.DeployStatus
	DeployHistoryPromise coaxer.Promise
}

// newDeployHistoryListBuilder creates a new DeployHistoryListBuilder bound to a
// particular singularity request.
func newDeployHistoryListBuilder(rc *requestContext) (*DeployHistoryListBuilder, error) {
	deployHistoryPromise := c.Coax(context.TODO(), func() (interface{}, error) {
		// We cannot expect to get Deploy details here, for that need to make a
		// separate request to GetDeploy.
		//
		// Get the last deploy for this request (count = 1, page = 1).
		return rc.Client.GetDeploys(rc.RequestID, 1, 1)

	}, "get deploy history for %q", rc.RequestID)
	return &DeployHistoryListBuilder{
		requestContext:       rc,
		DeployHistoryPromise: deployHistoryPromise,
	}, nil
}

// DeployHistory waits for the deploy history to be collected and then returns
// the result.
func (ds *DeployHistoryListBuilder) DeployHistory() (dtos.SingularityDeployHistoryList, error) {
	if err := ds.DeployHistoryPromise.Err(); err != nil {
		return nil, err
	}
	return ds.DeployHistoryPromise.Value().(dtos.SingularityDeployHistoryList), nil
}
