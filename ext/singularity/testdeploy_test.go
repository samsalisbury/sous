package singularity

import "github.com/opentable/go-singularity/dtos"

// A testDeploy represents a single deployment.
type testDeployHistory struct {
	DeployHistoryItem *dtos.SingularityDeployHistory
}

type newTestDeployHistoryParams struct {
	requestID, deployID, dockerImageName         string
	deployMarkerTimestamp, deployResultTimestamp int64
}

func newTestDeployHistory(params newTestDeployHistoryParams) *testDeployHistory {
	return &testDeployHistory{
		DeployHistoryItem: newSingularityDeployHistory(params),
	}

}
