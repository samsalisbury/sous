package singularity

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
)

func TestGetDepSetWorks(t *testing.T) {
	assert := assert.New(t)

	baseURL := "http://test-singularity.org"
	whip := make(map[string]swaggering.DummyControl)

	reg := sous.NewDummyRegistry()
	client, co := singularity.NewDummyClient(baseURL)
	co.FeedDTO(&dtos.SingularityRequestParentList{
		&dtos.SingularityRequestParent{
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

	co.FeedDTO(&dtos.SingularityDeployHistory{
		Deploy: &dtos.SingularityDeploy{
			Id: "testdep",
			ContainerInfo: &dtos.SingularityContainerInfo{
				Type:   dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
				Docker: &dtos.SingularityDockerInfo{},
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
	}, nil)
	whip[baseURL] = co

	dep := Deployer{
		Registry: reg,
		Client:   client,
		Cluster:  sous.Cluster{BaseURL: baseURL},
	}

	res, err := dep.RunningDeployments()
	assert.NoError(err)
	assert.NotNil(res)
}
