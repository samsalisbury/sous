package singularity

import (
	"context"
	"fmt"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/coaxer"
)

// DeployHistoryBuilder gathers information about the state of deployments.
type DeployHistoryBuilder struct {
	requestContext *requestContext
	// PreviousDeployID is the ID of the previous deploy for request
	// requestContext.RequestID.
	PreviousDeployID string
	// PreviousDeployStatus is the status of the previous deployment.
	PreviousDeployStatus sous.DeployStatus
	DeployHistoryPromise coaxer.Promise
}

// newDeploymentBuilder creates a new DeployStateBuilder bound to a particular sous deployment.
func newDeployHistoryBuilder(rc *requestContext) (*DeployHistoryBuilder, error) {
	deployHistoryPromise := c.Coax(context.TODO(), func() (interface{}, error) {
		//
		// We cannot expect to get Deploy details here, for that need to make a
		// separate request to GetDeploy.
		//
		// TODO: Make a request to GetDeploy to get the deploy details!
		// TODO: Ensure the test mock exhibits the same behaviour re providing
		//       no deploy field with these deploy history items.
		//

		// Get the last deploy for this request (count = 1, page = 1).
		deploys, err := rc.Client.GetDeploys(rc.RequestID, 1, 1)
		// Probable HTTP error, try again.
		if err != nil {
			return nil, err
		}
		// If there is no deploy history, try again.
		if len(deploys) == 0 {
			return nil, temporary{fmt.Errorf("no deploy history, trying again")}
		}
		// If the the deploy history returned has no deploy, then try again.
		if deploys[0].Deploy == nil {
			return nil, temporary{fmt.Errorf("deploy history has no deploy")}
		}
		return deploys, nil

	}, "get deploy history for %q", rc.RequestID)
	return &DeployHistoryBuilder{
		requestContext:       rc,
		DeployHistoryPromise: deployHistoryPromise,
	}, nil
}

// DeployHistory waits for the deploy history to be collected and then returns
// the result.
func (ds *DeployHistoryBuilder) DeployHistory() (dtos.SingularityDeployHistoryList, error) {
	if err := ds.DeployHistoryPromise.Err(); err != nil {
		return nil, err
	}
	return ds.DeployHistoryPromise.Value().(dtos.SingularityDeployHistoryList), nil
}
