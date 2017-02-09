package singularity

// XXX I got started with this, but it needs a dummy implementation of the
// singularity client, which needs extension of go-singularity and
// swagger-client-maker
import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
)

func TestBuildingRequestID(t *testing.T) {
	cn := "test-cluster"
	db := &deploymentBuilder{
		clusters: make(sous.Clusters),
		request:  &dtos.SingularityRequest{},
	}
	db.clusters[cn] = &sous.Cluster{}
	if err := db.assignClusterName(); err != nil {
		t.Errorf("unexpect error: %v", err)
	}
	if db.Deployment.Active.ClusterName != cn {
		t.Errorf("db.Target.ClusterName was %v expected %v", db.Deployment.Active.ClusterName, cn)
	}
}

func TestBuildDeployment_determineDeployStatus_missingstate(t *testing.T) {
	db := &deploymentBuilder{
		req: Request{},
	}

	if db.determineDeployStatus() == nil {
		t.Error("expected an error when deploy state missing")
	}

	db.req.RequestParent = &dtos.SingularityRequestParent{}

	if db.determineDeployStatus() == nil {
		t.Error("expected an error when request parent missing")
	}

}

func TestBuildDeployment_determineDeployStatus_pendingonly(t *testing.T) {
	depMarker := dtos.SingularityDeployMarker{}

	db := &deploymentBuilder{
		req: Request{
			RequestParent: &dtos.SingularityRequestParent{
				RequestDeployState: &dtos.SingularityRequestDeployState{
					PendingDeploy: &depMarker,
				},
			},
		},
	}

	if err := db.determineDeployStatus(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	} else {
		if db.Deployment.ActiveStatus != sous.DeployStatusPending {
			t.Errorf("Expected Status pending (%d), got %d", sous.DeployStatusPending, db.Deployment.ActiveStatus)
		}
		if db.depMarker != &depMarker {
			t.Errorf("Expected depMarker to be %v, got %v", depMarker, db.depMarker)
		}
	}

}

func TestBuildDeployment_determineDeployStatus_activeonly(t *testing.T) {
	depMarker := dtos.SingularityDeployMarker{}

	db := &deploymentBuilder{
		req: Request{
			RequestParent: &dtos.SingularityRequestParent{
				RequestDeployState: &dtos.SingularityRequestDeployState{
					ActiveDeploy: &depMarker,
				},
			},
		},
	}

	if err := db.determineDeployStatus(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	} else {
		if db.Deployment.ActiveStatus != sous.DeployStatusActive {
			t.Errorf("Expected Status pending (%d), got %d", sous.DeployStatusActive, db.Deployment.ActiveStatus)
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
		req: Request{
			RequestParent: &dtos.SingularityRequestParent{
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
		if db.Deployment.ActiveStatus != sous.DeployStatusPending {
			t.Errorf("Expected Status pending (%d), got %d", sous.DeployStatusPending, db.Deployment.ActiveStatus)
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
		clusters: clusters,
		request:  &dtos.SingularityRequest{Id: "::" + cn},
		req:      Request{URL: url},
	}
	assert.NoError(t, db.assignClusterName())
	assert.Equal(t, db.Deployment.Active.ClusterName, cn)

	db2 := &deploymentBuilder{
		clusters: clusters,
		request:  &dtos.SingularityRequest{Id: "::" + cn2},
		req:      Request{URL: url},
	}
	assert.NoError(t, db2.assignClusterName())
	assert.Equal(t, db2.Deployment.Active.ClusterName, cn2)
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
