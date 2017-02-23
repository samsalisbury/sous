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

// DeployStateBuilder gathers information about the state of deployments.
type DeployStateBuilder struct {
	*requestContext
	// PreviousDeployID is the ID of the previous deploy for request
	// requestContext.RequestID.
	PreviousDeployID string
	// PreviousDeployStatus is the status of the previous deployment.
	PreviousDeployStatus sous.DeployStatus
	DeployHistoryPromise coaxer.Promise
}

// newDeploymentBuilder creates a new DeployStateBuilder bound to a particular sous deployment.
func newDeployStateBuilder(rc *requestContext) (*DeployStateBuilder, error) {
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
	return &DeployStateBuilder{
		requestContext:       rc,
		DeployHistoryPromise: deployHistoryPromise,
	}, nil
}

func (ds *DeployStateBuilder) newDeploymentBuilder(deployID string) *DeploymentBuilder {
	return newDeploymentBuilder(ds.requestContext, deployID)
}

// DeployState returns the Sous deploy state.
func (ds *DeployStateBuilder) DeployState() (*sous.DeployState, error) {

	currentDeployID, currentDeployStatus, err := ds.currentDeployIDAndStatus()
	if err != nil {
		return nil, err
	}

	log.Printf("Gathering deploy state for current deploy %q; request %q", currentDeployID, ds.RequestID)

	currentDeployBuilder := ds.newDeploymentBuilder(currentDeployID)

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
	deployHistory, err := ds.DeployHistory()
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
		log.Printf("Gathering deploy state for last attempted deploy %q; request %q", lastDeployID, ds.RequestID)
		lastDeployBuilder = ds.newDeploymentBuilder(lastDeployID)
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

// DeployHistory waits for the deploy history to be collected and then returns
// the result.
func (ds *DeployStateBuilder) DeployHistory() (dtos.SingularityDeployHistoryList, error) {
	if err := ds.DeployHistoryPromise.Err(); err != nil {
		return nil, err
	}
	return ds.DeployHistoryPromise.Value().(dtos.SingularityDeployHistoryList), nil
}
