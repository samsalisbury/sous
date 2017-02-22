package singularity

import (
	"log"
	"testing"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
)

// defaultTestFixture is the starting point for all tests.
// Test modify this to make interesting assertions.
//
// It returns a testFixture for modification and a Deployer hooked up to use the
// fixture's Registry, DeployReaderFactory and Clusters.
func defaultTestFixture() (*testFixture, *Deployer) {
	fixture := &testFixture{
		Registry: newTestRegistry(),
	}

	singularity1 := fixture.AddSingularity("http://singularity1.com")
	singularity1.AddCluster("cluster1")

	// 1 cluster on singularity2.
	//cluster3 := fixture.AddCluster("cluster3", "http://singularity2.com")

	// Add 1 requests to cluster 1.
	cluster1Request1 := singularity1.AddRequestParent("github.com>user>repo1::cluster1", nil)
	cluster1Request2 := singularity1.AddRequestParent("github.com>user>repo2::cluster1", nil)

	// Add 2 successful deployments to cluster 1, request 1.
	cluster1Request1.AddStandardDeployHistory("deploy111", nil)
	cluster1Request1.AddStandardDeployHistory("deploy112", nil)

	// Add 1 successful deployment to cluster 1, request 2.
	cluster1Request2.AddStandardDeployHistory("deploy121", nil)

	return fixture, &Deployer{
		Registry:      fixture.Registry,
		ClientFactory: fixture.DeployReaderFactory,
		Clusters:      fixture.Clusters,
	}
}

type newRequestParentParams struct {
	requestID string
}

// newSingularityRequest is used as the base for all new singularity requests
// created with AddStandardRequestParent.
// It is in this file along with the tests for easy reference.
func newSingularityRequestParent(params newRequestParentParams) *dtos.SingularityRequestParent {
	return &dtos.SingularityRequestParent{
		// RequestDeployState is nil, reflecting Singularity's behaviour.
		RequestDeployState: nil,
		Request: &dtos.SingularityRequest{
			Id:          params.requestID,
			RequestType: dtos.SingularityRequestRequestTypeSERVICE,
			Instances:   3,
		},
		// Active is the default request state, it mostly means "not paused".
		// This is not to be confused with the state of the current deployment!
		State: dtos.SingularityRequestParentRequestStateACTIVE,
	}
}

// newSingularityDeployHistory is used to create all new deploy history items.
// It is in this file along with the tests for easy reference.
func newSingularityDeployHistory(params newTestDeployHistoryParams) *dtos.SingularityDeployHistory {
	return &dtos.SingularityDeployHistory{
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
	}
}

// defaultExpectedDeployState returns a sous.DeployState that corresponds
// with a default singularity request with a single default deployment.
// Note that the deployID parameter must parse to a sous.DeployID, which is
// distinct from a singularity request id, and from a singularity deploy id.
func defaultExpectedDeployState(deployID string, configure func(*sous.DeployState)) *sous.DeployState {
	did, err := sous.ParseDeployID(deployID)
	if err != nil {
		log.Panic(err)
	}
	ds := &sous.DeployState{
		Status: sous.DeployStatusSucceeded,
		Deployment: sous.Deployment{
			Kind: sous.ManifestKindService,
			SourceID: sous.SourceID{
				Location: did.ManifestID.Source,
				Version:  semv.MustParse("1"),
			},
			Flavor:      did.ManifestID.Flavor,
			ClusterName: did.Cluster,
			DeployConfig: sous.DeployConfig{
				NumInstances: 3, // From the SingularityRequest.
				Env: sous.Env{
					"TEST_ENV_VAR": "YES",
				},
				Resources: sous.Resources{
					"cpus":   "1.23",
					"memory": "123.45",
					"ports":  "1",
				},
				Volumes: sous.Volumes{
					&sous.Volume{
						Host:      "/host/path",
						Container: "/container/path",
						Mode:      sous.ReadWrite,
					},
				},
			},
		},
	}
	if configure != nil {
		configure(ds)
	}
	return ds
}

// defaultExpectedDeployStates returns the expected deploy states generated by
// defaultTextFixture.
func defaultExpectedDeployStates() sous.DeployStates {
	return sous.NewDeployStates(
		defaultExpectedDeployState("github.com/user/repo1:cluster1", nil),
		defaultExpectedDeployState("github.com/user/repo2:cluster1", nil),
	)
}

// TestDeployer_RunningDeployments tests entire groups of clusters, running on
// multiple singularities using short test cases.
//
// In order to hide the complexity of such huge data structures (which otherwise
// drown out the meaning of the test) we adopt the following strategy:
//
// 1. Start with a pre-configured "default" input test fixture.
//    This input has already been configured to look like a somewhat realistic
//    Singularity state.
// 2. Start also with a pre-configured expected output sous.DeployStates.
// 3. First, assert that the pre-configured input results in the pre-configured
//    expected output.
// 4. For each assertion, modify the provided input in some way, and also modify
//    the expected output congruously. Thus we can assert that the difference in
//    input resulted in the corresponding difference in output.
func TestDeployer_RunningDeployments(t *testing.T) {

	testCases := []struct {
		// Desc describes the input and expected output.
		Description string
		// InputModifier is called on the result of defaultTestFixture before
		// RunningDeployments is called on the group of clusters it describes.
		InputModifier InputModifier
		// ExpectedModifier is called on the result of
		// defaultExpectedDeployStates before running assertions.
		ExpectedModifier ExpectedModifier
	}{
		{
			"Unmodified default input => unmodified default expected output",
			func(*testFixture) {
				// Do nothing.
			},
			func(*sous.DeployStates) {
				// Do nothing.
			},
		},
		{
			"Latest deploy pending => DeployStatusPending",
			modifyInputRequestParent("http://singularity1.com", "github.com>user>repo1::cluster1",
				func(request *testRequestParent) {
					// Add a new pending deployment.
					request.AddStandardDeployHistory("newDeploy", func(d *dtos.SingularityDeployHistory) {
						d.DeployResult.DeployState = dtos.SingularityDeployResultDeployStateWAITING
					})
				}),
			modifyExpectedDeployState("github.com/user/repo1:cluster1",
				func(ds *sous.DeployState) {
					// Expect the deploy state to be pending.
					ds.Status = sous.DeployStatusPending
				}),
		},
		{
			"Latest deploy failed => DeployStatusFailed",
			modifyInputRequestParent("http://singularity1.com", "github.com>user>repo1::cluster1",
				func(request *testRequestParent) {
					// Add a new failed deployment.
					request.AddStandardDeployHistory("newDeploy", func(d *dtos.SingularityDeployHistory) {
						d.DeployResult.DeployState = dtos.SingularityDeployResultDeployStateFAILED
					})
				}),
			modifyExpectedDeployState("github.com/user/repo1:cluster1",
				func(ds *sous.DeployState) {
					// Expect the deploy state to be failed.
					ds.Status = sous.DeployStatusFailed
				}),
		},
	}

	// Run the test cases.
	for _, test := range testCases {
		test := test
		t.Run(test.Description, func(t *testing.T) {
			// Set up the input.
			fixture, deployer := defaultTestFixture()
			test.InputModifier(fixture)

			// Set up expectations.
			expected := defaultExpectedDeployStates()
			test.ExpectedModifier(&expected)

			// Get the actual output.
			actual, err := deployer.RunningDeployments()
			if err != nil {
				// These tests are only concerned with non-error states.
				t.Fatal(err)
			}

			// Assert actual == expected.
			different, diffs := actual.Diff2(expected)
			if !different {
				return // Success!
			}
			for _, d := range diffs {
				t.Error(d)
			}
		})
	}
}

type InputModifier func(*testFixture)
type ExpectedModifier func(*sous.DeployStates)

func modifyInputRequestParent(singularityBaseURL, requestID string, modifyRequestParent func(*testRequestParent)) InputModifier {
	return func(fixture *testFixture) {
		singularity, ok := fixture.Singularities[singularityBaseURL]
		if !ok {
			log.Panicf("No singularity called %q", singularityBaseURL)
		}
		request, ok := singularity.Requests[requestID]
		if !ok {
			log.Panicf("Singularity %q contains no request %q", singularityBaseURL, requestID)
		}
		modifyRequestParent(request)
	}
}

func modifyExpectedDeployState(sousDeployID string, modifyDeployState func(*sous.DeployState)) ExpectedModifier {
	did := sous.MustParseDeployID(sousDeployID)
	return func(deployStates *sous.DeployStates) {
		deployState, ok := deployStates.Get(did)
		if !ok {
			log.Panicf("No deploy ID called %q", did)
		}
		// Modify and re-set the deploy state as that doesn't rely on it being a
		// pointer.
		modifyDeployState(deployState)
		deployStates.Set(deployState.ID(), deployState)
	}
}
