package singularity

import "github.com/opentable/go-singularity/dtos"

// A testDeploy represents a single deployment.
type testDeploy struct {
	DeployHistoryItem *dtos.SingularityDeployHistory
}

type newTestDeployHistoryParams struct {
	requestID, deployID, dockerImageName         string
	deployMarkerTimestamp, deployResultTimestamp int64
}

func newTestDeployHistory(params newTestDeployHistoryParams) *testDeploy {
	return &testDeploy{
		DeployHistoryItem: &dtos.SingularityDeployHistory{
			Deploy: &dtos.SingularityDeploy{
				Id: params.deployID,
				ContainerInfo: &dtos.SingularityContainerInfo{
					Type: dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
					Docker: &dtos.SingularityDockerInfo{
						// TODO: Other docker config.
						Image: params.dockerImageName,
					},
					Volumes: dtos.SingularityVolumeList{
						&dtos.SingularityVolume{
							HostPath:      "/host/path",
							ContainerPath: "/container/path",
							Mode:          dtos.SingularityVolumeSingularityDockerVolumeModeRW,
						},
					},
				},
				Resources: &dtos.Resources{
					Cpus:     1.23,
					MemoryMb: 123.45,
					NumPorts: 1,
				},
				Env: map[string]string{
					"TEST_ENV_VAR": "YES",
				},
			},
			DeployResult: &dtos.SingularityDeployResult{
				// Successful deploy result by default.
				DeployState: dtos.SingularityDeployResultDeployStateSUCCEEDED,
				// DeployFailures is not nil in Singularity, it's an empty array.
				DeployFailures: dtos.SingularityDeployFailureList{},
				Timestamp:      params.deployResultTimestamp,
			},
			DeployMarker: &dtos.SingularityDeployMarker{
				RequestId: params.requestID,
				DeployId:  params.deployID,
				Timestamp: params.deployMarkerTimestamp,
				User:      "some user",
			},
		},
	}

}
