package singularity

// XXX I got started with this, but it needs a dummy implementation of the
// singularity client, which needs extension of go-singularity and
// swagger-client-maker
import (
	"testing"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/swaggering"
	"github.com/stretchr/testify/assert"
)

type (
	fakeImageLabeller struct {
		cannedAnswer map[string]string
	}
)

func (fake *fakeImageLabeller) ImageLabels(imageName string) (labels map[string]string, err error) {
	return fake.cannedAnswer, nil
}

func TestBuildDeployment_errors(t *testing.T) {
	url := "http://example.com/singularity"
	reqParent := &dtos.SingularityRequestParent{
		RequestDeployState: &dtos.SingularityRequestDeployState{},
	}
	testClusters := sous.Clusters{
		"left":  &sous.Cluster{Name: "left", BaseURL: url},
		"right": &sous.Cluster{Name: "right", BaseURL: url},
	}
	fakeSing, c := newSingClientSpy()

	cannedRequest := &dtos.SingularityRequestParent{
		RequestDeployState: &dtos.SingularityRequestDeployState{},
	}
	c.cannedRequest(cannedRequest)

	cannedDep := &dtos.SingularityDeployHistory{}
	c.cannedDeploy(cannedDep)

	fakeReg := &fakeImageLabeller{
		cannedAnswer: map[string]string{},
	}

	log, _ := logging.NewLogSinkSpy()

	req := SingReq{
		SourceURL: url,
		Sing:      fakeSing,
		ReqParent: reqParent,
	}
	_, err := BuildDeployment(fakeReg, testClusters, req, log)

	assert.Error(t, err)

	req.ReqParent.RequestDeployState = &dtos.SingularityRequestDeployState{}
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	req.ReqParent.Request = &dtos.SingularityRequest{Id: "1234"}
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	req.ReqParent.RequestDeployState.ActiveDeploy = &dtos.SingularityDeployMarker{}
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	cannedDep.Deploy = &dtos.SingularityDeploy{}
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	cannedDep.Deploy.ContainerInfo = &dtos.SingularityContainerInfo{}
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	cannedDep.Deploy.ContainerInfo.Type = "DOCKER"
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	cannedDep.Deploy.ContainerInfo.Docker = &dtos.SingularityDockerInfo{Image: "image-name"}
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	fakeReg.cannedAnswer["com.opentable.sous.repo_url"] = "repo_url"
	fakeReg.cannedAnswer["com.opentable.sous.version"] = "version"
	fakeReg.cannedAnswer["com.opentable.sous.revision"] = "revision"
	fakeReg.cannedAnswer["com.opentable.sous.repo_offset"] = "repo_offset"
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	fakeReg.cannedAnswer["com.opentable.sous.version"] = "1.2.3"
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	req.ReqParent.Request.Id = "repo_url,repo_offset::left"
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	cannedDep.Deploy.Metadata = map[string]string{
		"com.opentable.sous.clustername": "left",
		"com.opentable.sous.flavor":      "vanilla",
	}

	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	cannedDep.Deploy.Resources = &dtos.Resources{}
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	req.ReqParent.Request.RequestType = dtos.SingularityRequestRequestTypeSERVICE
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.Error(t, err)

	cannedDep.DeployMarker = &dtos.SingularityDeployMarker{}
	_, err = BuildDeployment(fakeReg, testClusters, req, log)
	assert.NoError(t, err)
}

func TestBuildDeployment(t *testing.T) {
	url := "http://example.com/singularity"
	testClusters := sous.Clusters{
		"left":  &sous.Cluster{Name: "left", BaseURL: url},
		"right": &sous.Cluster{Name: "right", BaseURL: url},
	}

	req := SingReq{
		SourceURL: url,
		ReqParent: &dtos.SingularityRequestParent{
			RequestDeployState: &dtos.SingularityRequestDeployState{
				ActiveDeploy: &dtos.SingularityDeployMarker{},
			},
			Request: &dtos.SingularityRequest{
				Id:          "repo_url,repo_offset::left",
				RequestType: dtos.SingularityRequestRequestTypeSERVICE,
				Owners:      swaggering.StringList{"jlester@opentable.com"},
			},
		},
	}

	log, _ := logging.NewLogSinkSpy()

	cannedRequest := &dtos.SingularityRequestParent{
		RequestDeployState: &dtos.SingularityRequestDeployState{},
	}

	cannedDep := &dtos.SingularityDeployHistory{
		DeployResult: &dtos.SingularityDeployResult{
			DeployState: dtos.SingularityDeployResultDeployStateSUCCEEDED,
		},
		DeployMarker: &dtos.SingularityDeployMarker{},
		Deploy: &dtos.SingularityDeploy{
			Metadata: map[string]string{
				"com.opentable.sous.clustername": "left",
				"com.opentable.sous.flavor":      "vanilla",
			},

			Healthcheck: &dtos.HealthcheckOptions{
				Uri:                    "/health-report",
				ResponseTimeoutSeconds: 350,
				StartupTimeoutSeconds:  700,
			},

			ContainerInfo: &dtos.SingularityContainerInfo{
				Type:   "DOCKER",
				Docker: &dtos.SingularityDockerInfo{Image: "image-name"},
				Volumes: dtos.SingularityVolumeList{
					&dtos.SingularityVolume{
						HostPath:      "hostpath",
						ContainerPath: "containerpath",
						Mode:          dtos.SingularityVolumeSingularityDockerVolumeModeRW,
					},
				},
			},
			Resources: &dtos.Resources{},
		},
	}
	fakeSing, fsc := newSingClientSpy()

	fsc.cannedRequest(cannedRequest)
	fsc.cannedDeploy(cannedDep)

	req.Sing = fakeSing

	fakeReg := &fakeImageLabeller{
		cannedAnswer: map[string]string{
			"com.opentable.sous.repo_url":    "repo_url",
			"com.opentable.sous.revision":    "revision",
			"com.opentable.sous.repo_offset": "repo_offset",
			"com.opentable.sous.version":     "1.2.3",
		},
	}

	actual, err := BuildDeployment(fakeReg, testClusters, req, log)

	assert.NoError(t, err)

	expected := sous.DeployState{Status: sous.DeployStatusActive}
	expected.ClusterName = "left"
	expected.Flavor = "vanilla"

	assert.Equal(t, actual.ClusterName, expected.ClusterName)
	assert.Equal(t, actual.Status, expected.Status)
	assert.Equal(t, actual.Flavor, expected.Flavor)

	assert.Equal(t, actual.Startup.CheckReadyURIPath, "/health-report")

	assert.Equal(t, actual.Startup.CheckReadyURITimeout, 350)

	assert.Equal(t, actual.Startup.Timeout, 700)
}

func TestBuildDeployment_failed_deploy(t *testing.T) {
	url := "http://example.com/singularity"
	testClusters := sous.Clusters{
		"left":  &sous.Cluster{Name: "left", BaseURL: url},
		"right": &sous.Cluster{Name: "right", BaseURL: url},
	}

	req := SingReq{
		SourceURL: url,
		ReqParent: &dtos.SingularityRequestParent{
			State: dtos.SingularityRequestParentRequestStateSYSTEM_COOLDOWN,

			RequestDeployState: &dtos.SingularityRequestDeployState{
				ActiveDeploy: &dtos.SingularityDeployMarker{},
			},
			Request: &dtos.SingularityRequest{
				Id:          "repo_url,repo_offset::left",
				RequestType: dtos.SingularityRequestRequestTypeSERVICE,
				Owners:      swaggering.StringList{"jlester@opentable.com"},
			},
		},
	}

	cannedRequest := &dtos.SingularityRequestParent{
		RequestDeployState: &dtos.SingularityRequestDeployState{},
	}

	cannedDep := &dtos.SingularityDeployHistory{
		//DEPLOY_RESULT=$(jq -r .deployResult.deployState < $DEPLOY_STATE)
		//$DEPLOY_RESULT = SUCCEEDED
		DeployResult: &dtos.SingularityDeployResult{
			DeployState: dtos.SingularityDeployResultDeployStateFAILED,
		},
		DeployMarker: &dtos.SingularityDeployMarker{},
		Deploy: &dtos.SingularityDeploy{
			Metadata: map[string]string{
				"com.opentable.sous.clustername": "left",
				"com.opentable.sous.flavor":      "",
			},
			ContainerInfo: &dtos.SingularityContainerInfo{
				Type:   "DOCKER",
				Docker: &dtos.SingularityDockerInfo{Image: "image-name"},
				Volumes: dtos.SingularityVolumeList{
					&dtos.SingularityVolume{
						HostPath:      "hostpath",
						ContainerPath: "containerpath",
						Mode:          dtos.SingularityVolumeSingularityDockerVolumeModeRW,
					},
				},
			},
			Resources: &dtos.Resources{},
		},
	}

	fakeSing, fsc := newSingClientSpy()
	fsc.cannedRequest(cannedRequest)
	fsc.cannedDeploy(cannedDep)

	req.Sing = fakeSing

	log, _ := logging.NewLogSinkSpy()

	fakeReg := &fakeImageLabeller{
		cannedAnswer: map[string]string{
			"com.opentable.sous.repo_url":    "repo_url",
			"com.opentable.sous.revision":    "revision",
			"com.opentable.sous.repo_offset": "repo_offset",
			"com.opentable.sous.version":     "1.2.3",
		},
	}

	actual, err := BuildDeployment(fakeReg, testClusters, req, log)

	assert.NoError(t, err)

	expected := sous.DeployState{Status: sous.DeployStatusFailed}
	expected.ClusterName = "left"

	assert.Equal(t, actual.ClusterName, expected.ClusterName)
	assert.Equal(t, actual.Status, expected.Status, "Expected %s got %s", expected.Status, actual.Status)
}

func TestBuildingRequestID(t *testing.T) {
	cn := "test-cluster"
	db := &deploymentBuilder{
		clusters: make(sous.Clusters),
		request:  &dtos.SingularityRequest{},
		deploy: &dtos.SingularityDeploy{
			Metadata: map[string]string{
				"com.opentable.sous.clustername": "test-cluster",
			},
		},
	}
	db.clusters[cn] = &sous.Cluster{}
	if err := db.restoreFromMetadata(); err != nil {
		t.Errorf("unexpect error: %v", err)
	}
	if db.Target.ClusterName != cn {
		t.Errorf("db.Target.ClusterName was %v expected %v", db.Target.ClusterName, cn)
	}
}

func TestBuildDeployment_determineDeployStatus_missingstate(t *testing.T) {
	db := &deploymentBuilder{
		req: SingReq{},
	}

	if db.determineDeployStatus() == nil {
		t.Error("expected an error when deploy state missing")
	}

	db.req.ReqParent = &dtos.SingularityRequestParent{}

	if db.determineDeployStatus() == nil {
		t.Error("expected an error when request parent missing")
	}

}

func TestBuildDeployment_determineDeployStatus_pendingonly(t *testing.T) {
	depMarker := dtos.SingularityDeployMarker{}

	db := &deploymentBuilder{
		req: SingReq{
			ReqParent: &dtos.SingularityRequestParent{
				State: dtos.SingularityRequestParentRequestStateACTIVE,
				RequestDeployState: &dtos.SingularityRequestDeployState{
					PendingDeploy: &depMarker,
				},
			},
		},
	}

	if err := db.determineDeployStatus(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	} else {
		if db.Target.Status != sous.DeployStatusPending {
			t.Errorf("Expected Status pending (%d), got %d", sous.DeployStatusPending, db.Target.Status)
		}
		if db.depMarker != &depMarker {
			t.Errorf("Expected depMarker to be %v, got %v", depMarker, db.depMarker)
		}
	}

}

func TestBuildDeployment_determineDeployStatus_activeAndPending(t *testing.T) {
	depMarker := dtos.SingularityDeployMarker{}
	otherDepMarker := dtos.SingularityDeployMarker{}

	db := &deploymentBuilder{
		req: SingReq{
			ReqParent: &dtos.SingularityRequestParent{
				State: dtos.SingularityRequestParentRequestStateACTIVE,
				RequestDeployState: &dtos.SingularityRequestDeployState{
					PendingDeploy: &depMarker,
					ActiveDeploy:  &otherDepMarker,
				},
			},
		},
	}

	if err := db.determineDeployStatus(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	} else {
		if db.Target.Status != sous.DeployStatusPending {
			t.Errorf("Expected Status pending (%d), got %d", sous.DeployStatusPending, db.Target.Status)
		}
		if db.depMarker != &depMarker {
			t.Errorf("Expected depMarker to be %v, got %v", depMarker, db.depMarker)
		}
	}

}
func TestBuildingRequestIDTwoClusters(t *testing.T) {
	cn := "test-cluster"
	cn2 := "test-other"
	url := "https://a.singularity.somewhere"
	clusters := make(sous.Clusters)
	clusters[cn] = &sous.Cluster{BaseURL: url}
	clusters[cn2] = &sous.Cluster{BaseURL: url}

	db := &deploymentBuilder{
		deploy: &dtos.SingularityDeploy{
			Metadata: map[string]string{
				"com.opentable.sous.clustername": cn,
			},
		},
		clusters: clusters,
		request:  &dtos.SingularityRequest{Id: "::" + cn},
		req:      SingReq{SourceURL: url},
	}
	assert.NoError(t, db.restoreFromMetadata())
	assert.Equal(t, db.Target.ClusterName, cn)

	db2 := &deploymentBuilder{
		deploy: &dtos.SingularityDeploy{
			Metadata: map[string]string{
				"com.opentable.sous.clustername": cn2,
			},
		},

		clusters: clusters,
		request:  &dtos.SingularityRequest{Id: "::" + cn2},
		req:      SingReq{SourceURL: url},
	}
	assert.NoError(t, db2.restoreFromMetadata())
	assert.Equal(t, db2.Target.ClusterName, cn2)
}

/*
func TestConstructDeployment(t *testing.T) {
	assert := assert.New(t)

	im := NewDummyNameCache()
	cl := NewDummyRectificationClient(im)
	req := singReq{
		sourceURL: "source.url",
		sing:      &DummyClient{}, //XXX need a dummy client for singularity
		reqParent: &dtos.SingularityRequestParent{
			Request:            &dtos.SingularityRequest{},
			RequestDeployState: &dtos.SingularityRequestDeployState{},
			ActiveDeploy:       &dtos.SingularityDeploy{},
			PendingDeploy:      &dtos.SingularityDeploy{},

			//			ExpiringBounce           *SingularityExpiringBounce           `json:"expiringBounce"`
			//			ExpiringPause            *SingularityExpiringPause            `json:"expiringPause"`
			//			ExpiringScale            *SingularityExpiringScale            `json:"expiringScale"`
			//			ExpiringSkipHealthchecks *SingularityExpiringSkipHealthchecks `json:"expiringSkipHealthchecks"`
			//			PendingDeployState       *SingularityPendingDeploy            `json:"pendingDeployState"`
			//			State                    SingularityRequestParentRequestState `json:"state"`
		},
	}

	//func assembleDeployment(cl RectificationClient, req singReq) (*Deployment, error) {
	uc := newDeploymentBuilder(cl, req)
	err := uc.completeConstruction()
	if assert.NoError(err) {
		if assert.Len(uc.target.DeployConfig.Volumes, 1) {
			assert.Equal("RO", string(uc.target.DeployConfig.Volumes[0].Mode))
		}
	}
}
*/
