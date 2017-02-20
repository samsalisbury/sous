package singularity

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
	"github.com/samsalisbury/semv"
)

func TestDeployer_RunningDeployments(t *testing.T) {

}

func TestGetDepSetWorks(t *testing.T) {
	assert := assert.New(t)

	const baseURL = "http://test-singularity.org/"
	const requestID = "github.com>user>project::cluster1"
	const deployID = "deploy1"
	const repo = "github.com/user/project"

	reg := sous.NewDummyRegistry()

	reg.FeedImageLabels(map[string]string{
		"com.opentable.sous.repo_url":    repo,
		"com.opentable.sous.version":     "1.0.0",
		"com.opentable.sous.revision":    "abc123",
		"com.opentable.sous.repo_offset": "",
	}, nil)

	testReq := &dtos.SingularityRequestParent{
		RequestDeployState: &dtos.SingularityRequestDeployState{
			ActiveDeploy: &dtos.SingularityDeployMarker{
				DeployId:  deployID,
				RequestId: requestID,
			},
		},
		Request: &dtos.SingularityRequest{
			Id:          requestID,
			RequestType: dtos.SingularityRequestRequestTypeSERVICE,
			Owners:      swaggering.StringList{"jlester@opentable.com"},
		},
	}

	testDep := &dtos.SingularityDeployHistory{
		Deploy: &dtos.SingularityDeploy{
			Id: deployID,
			ContainerInfo: &dtos.SingularityContainerInfo{
				Type: dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
				Docker: &dtos.SingularityDockerInfo{
					Image: "some-docker-image",
				},
				Volumes: dtos.SingularityVolumeList{
					&dtos.SingularityVolume{
						HostPath:      "/onhost",
						ContainerPath: "/indocker",
						Mode:          dtos.SingularityVolumeSingularityDockerVolumeModeRW,
					},
				},
			},
			Resources: &dtos.Resources{},
		},
	}

	requester := TestGETRequester{}
	requester.RegisterDTO(&dtos.SingularityRequestParentList{testReq}, "/api/requests")
	requester.RegisterDTO(testReq, "/api/requests/request/%s", requestID)
	requester.RegisterDTO(testDep, "/api/history/request/%s/deploy/%s", requestID, deployID)

	client := &singularity.Client{Requester: requester}

	dep := Deployer{
		Registry:      reg,
		ClientFactory: func(*sous.Cluster) *singularity.Client { return client },
		Clusters:      sous.Clusters{"cluster1": &sous.Cluster{Name: "cluster1", BaseURL: baseURL}},
	}

	res, err := dep.RunningDeployments()

	if !assert.NoError(err) {
		t.FailNow()
	}

	if !assert.NotNil(res) {
		t.FailNow()
	}

	actual := res.Snapshot()

	if !assert.Len(actual, 1) {
		t.FailNow()
	}

	t.Logf("% #v", res.Snapshot())

	expectedDID := sous.DeployID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: repo,
				Dir:  "",
			},
			Flavor: "",
		},
		Cluster: "cluster1",
	}

	actualDS, ok := actual[expectedDID]
	if !ok {
		var actualDID sous.DeployID
		for actualDID = range actual {
			break
		}
		t.Fatalf("Got DeployID %q; want DeployID %q", actualDID, expectedDID)
	}

	expectedDS := sous.DeployState{
		Deployment: sous.Deployment{
			Kind: sous.ManifestKindService,
			SourceID: sous.SourceID{
				Location: sous.SourceLocation{
					Repo: repo,
					Dir:  "",
				},
				Version: semv.MustParse("1"),
			},
			Flavor:      "",
			ClusterName: "cluster1",
			//Cluster:     cluster,
			DeployConfig: sous.DeployConfig{
				Resources: sous.Resources{
					"cpus":   "0",
					"memory": "0",
					"ports":  "0",
				},
				Volumes: sous.Volumes{
					&sous.Volume{
						Host:      "/onhost",
						Container: "/indocker",
						Mode:      sous.ReadWrite,
					},
				},
			},
		},
		Status: sous.DeployStatusActive,
	}

	if different, diffs := actualDS.Diff(&expectedDS); different {
		t.Fatalf("deploy state not as expected: % #v", diffs)
	}

}
