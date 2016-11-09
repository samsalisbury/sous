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
	if db.Target.ClusterName != cn {
		t.Errorf("db.Target.ClusterName was %v expected %v", db.Target.ClusterName, cn)
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
		req:      SingReq{SourceURL: url},
	}
	assert.NoError(t, db.assignClusterName())
	assert.Equal(t, db.Target.ClusterName, cn)

	db2 := &deploymentBuilder{
		clusters: clusters,
		request:  &dtos.SingularityRequest{Id: "::" + cn2},
		req:      SingReq{SourceURL: url},
	}
	assert.NoError(t, db2.assignClusterName())
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
