package singularity

import "github.com/opentable/go-singularity/dtos"

// A testRequest represents all the request-scoped data for a single
// singularity request.
//
// It provides functions that make it easy to construct a consistent
// milieu in which tests can be run. The strategy for writing tests
// with this is to construct a healthy and consistent world, and then
// to introduce specific changes against which tests can be written.
type testRequest struct {
	Parent        *testSingularity
	RequestParent *dtos.SingularityRequestParent
	// Error to be returned instead of RequestParent.
	Error   error
	Deploys map[string]*testDeployHistory
}

// AddDeploy adds a new DeployHistory linked with this request. The configure
// func is called on it to manipulate it before it's added to the deploy history
// and returned wrapped in a testDeploy.
//
// The added deployment will have a timestamp at least one more than the last.
//
// AddDeploy also adds:
//   - A corresponding docker image to the test registry owned
//     by the ancestor testFixture (at Parent.Parent.Parent)
//   - A corresponding entry in SingularityRequestDeployState if the
//     status is Pending or Active after configure is called.
func (tr *testRequest) AddStandardDeployHistory(deployID string, configure func(*dtos.SingularityDeployHistory)) *testDeployHistory {
	if tr.Deploys == nil {
		tr.Deploys = map[string]*testDeployHistory{}
	}

	// Derive data needed to create the singularity deploy history item.
	requestID := tr.RequestParent.Request.Id // this is used a few times.

	// Add docker image to the test registry.
	dockerImageName := tr.Parent.Parent.Registry.AddImage(requestID, "1.0.0")

	// Get some timestamps. The order here mimics observed Singularity behaviour
	// where deploy markers always have timestamps earlier than deploy results.
	deployMarkerTimestamp := nextDeployTimestamp()
	deployResultTimestamp := nextDeployTimestamp()

	// Create a new deploy history item.
	deployHistory := newTestDeployHistory(newTestDeployHistoryParams{
		requestID:             requestID,
		deployID:              deployID,
		dockerImageName:       dockerImageName,
		deployMarkerTimestamp: deployMarkerTimestamp,
		deployResultTimestamp: deployResultTimestamp,
	})

	// All defaults are set, now pass the deploy to provided configure func.
	if configure != nil {
		configure(deployHistory.DeployHistoryItem)
	}

	tr.AddDeployHistory(deployHistory)

	return deployHistory
}

// AddDeployHistory adds a deploy to the history and updates the request to
// reflect this deployment.
func (tr *testRequest) AddDeployHistory(testDeployHistory *testDeployHistory) {
	deployHistory := testDeployHistory.DeployHistoryItem
	// Configure the request to reflect this latest deploy.
	deployMarkerCopy := *deployHistory.DeployMarker
	switch deployHistory.DeployResult.DeployState {
	default:
		// The default case represents all failure modes. Singularity does not
		// change the active deploy to a failed one (so leave the last one in
		// place. However, if there was a pending deployment with the same
		// deploy ID, it will be removed from PendingDeploy.
		oldPending := tr.RequestParent.RequestDeployState.PendingDeploy
		if oldPending != nil && oldPending.DeployId == deployHistory.Deploy.Id {
			tr.RequestParent.RequestDeployState.PendingDeploy = nil
		}
	case dtos.SingularityDeployResultDeployStateSUCCEEDED:
		// SUCCEEDED, set Active deploy.
		tr.RequestParent.RequestDeployState.ActiveDeploy = &deployMarkerCopy
	case dtos.SingularityDeployResultDeployStateWAITING:
		// WAITING, set Pending deploy.
		tr.RequestParent.RequestDeployState.PendingDeploy = &deployMarkerCopy
	}

	tr.Deploys[deployHistory.Deploy.Id] = testDeployHistory
}
