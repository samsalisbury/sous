package singularity

import (
	"testing"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/swaggering"
	"github.com/stretchr/testify/assert"
)

func TestDeployer_RunningDeployments(t *testing.T) {
	assert := assert.New(t)

	whip := make(map[string]swaggering.DummyControl)

	reg := sous.NewDummyRegistry()
	client := sous.NewDummyRectificationClient()
	singFac := func(url string) singClient {
		cl, co := singularity.NewDummyClient(url)

		co.FeedDTO(&dtos.SingularityRequestParentList{
			&dtos.SingularityRequestParent{
				State: dtos.SingularityRequestParentRequestStateACTIVE,
				RequestDeployState: &dtos.SingularityRequestDeployState{
					ActiveDeploy: &dtos.SingularityDeployMarker{
						DeployId:  "testdep",
						RequestId: "testreq",
					},
				},
				Request: &dtos.SingularityRequest{
					Id:          "testreq",
					RequestType: dtos.SingularityRequestRequestTypeSERVICE,
					Owners:      swaggering.StringList{"jlester@opentable.com"},
				},
			},
		}, nil)

		co.FeedDTO(&dtos.SingularityRequestParent{
			State: dtos.SingularityRequestParentRequestStateACTIVE,
			RequestDeployState: &dtos.SingularityRequestDeployState{
				ActiveDeploy: &dtos.SingularityDeployMarker{
					DeployId:  "testdep",
					RequestId: "testreq",
				},
			},
			Request: &dtos.SingularityRequest{
				Id:          "testreq",
				RequestType: dtos.SingularityRequestRequestTypeSERVICE,
				Owners:      swaggering.StringList{"jlester@opentable.com"},
			},
		}, nil)

		item := &dtos.SingularityDeployHistory{
			DeployResult: &dtos.SingularityDeployResult{
				DeployState: "ACTIVE",
			},
			DeployMarker: &dtos.SingularityDeployMarker{
				RequestId: "testreq",
				DeployId:  "testdep",
			},
			Deploy: &dtos.SingularityDeploy{
				User: "",
				Metadata: map[string]string{
					"": "",
				},
				Id:        "testdep",
				RequestId: "testreq",
			},
		}
		// TODO SS: Add this item to request history to make test more complete.
		// Right now the assertions are low in value, just looking for nil error
		// and non-nil other response.
		//
		// We currently have an integration test failure, but there is no easy
		// way to cover the case in this unit test.
		print(item)
		co.FeedDTO(&dtos.SingularityDeployHistoryList{}, nil)

		whip[url] = co
		return cl
	}
	dep := deployer{
		Client:        client,
		singFac:       singFac,
		ReqsPerServer: 10,
		log:           logging.SilentLogSet(),
	}

	clusters := sous.Clusters{"test": {BaseURL: "http://test-singularity.org/"}}
	res, err := dep.RunningDeployments(reg, clusters)
	assert.NoError(err)
	assert.NotNil(res)
	// TODO SS: Add more assertions here once we have the test returning
	// some deployments - right now the line below always prints:
	//   Num running: 0
	t.Logf("Num running: %d", len(res.Snapshot()))
}
