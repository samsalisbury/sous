package singularity

import (
	"log"

	"github.com/opentable/go-singularity/dtos"
)

// A testRequest represents all the request-scoped data for a single
// singularity request.
//
// It provides functions that make it easy to construct a consistent
// milieu in which tests can be run. The strategy for writing tests
// with this is to construct a healthy and consistent world, and then
// to introduce specific flaws against which tests can be written.
type testRequest struct {
	Parent        *testSingularity
	RequestParent *dtos.SingularityRequestParent
	// Error to be returned instead of RequestParent.
	Error   error
	Deploys map[string]*testDeploy
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
func (tr *testRequest) AddDeploy(deployID string, configure func(*dtos.SingularityDeployHistory)) *testDeploy {
	if tr.Deploys == nil {
		tr.Deploys = map[string]*testDeploy{}
	}

	// Derive data needed to create the singularity deploy history item.
	requestID := tr.RequestParent.Request.Id // this is used a few times.
	did, err := ParseRequestID(requestID)
	if err != nil {
		log.Fatal(err)
	}

	// Calculate test docker image name.
	repo := did.ManifestID.Source.Repo
	offset := did.ManifestID.Source.Dir
	tag := "1.0.0"
	dockerImageName := testImageName(repo, offset, tag)
	// Add docker image to the test registry.
	tr.Parent.Parent.Registry.AddImage(dockerImageName, repo, offset, tag)

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
	}).DeployHistoryItem

	// All defaults are set, now pass the deploy to provided configure func.
	if configure != nil {
		configure(deployHistory)
	}

	// Configure the request to reflect this latest deploy if it was successful
	// or pending. Other statuses may be important but are not currently
	// reflected.
	deployMarkerCopy := *deployHistory.DeployMarker
	switch deployHistory.DeployResult.DeployState {
	case dtos.SingularityDeployResultDeployStateSUCCEEDED:
		// SUCCEEDED, set Active deploy.
		tr.RequestParent.RequestDeployState.ActiveDeploy = &deployMarkerCopy
	case dtos.SingularityDeployResultDeployStateWAITING:
		// WAITING, set Pending deploy.
		tr.RequestParent.RequestDeployState.PendingDeploy = &deployMarkerCopy
	}

	// Add the deploy history to this testRequest.
	deploy := &testDeploy{
		DeployHistoryItem: deployHistory,
	}
	tr.Deploys[deployHistory.Deploy.Id] = deploy

	return deploy
}