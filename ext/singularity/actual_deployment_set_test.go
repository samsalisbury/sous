package singularity

import (
	"testing"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
	"github.com/stretchr/testify/assert"
)

func TestGetDepSetWorks(t *testing.T) {
	assert := assert.New(t)

	whip := make(map[string]swaggering.DummyControl)

	reg := sous.NewDummyRegistry()
	client := NewDummyRectificationClient(reg)
	dep := deployer{client, reg,
		func(url string) *singularity.Client {
			cl, co := singularity.NewDummyClient(url)

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
					},
				},
			}, nil)

			co.FeedDTO(&dtos.SingularityDeployHistory{
				Deploy: &dtos.SingularityDeploy{
					Id: "testdep",
					ContainerInfo: &dtos.SingularityContainerInfo{
						Type:   dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
						Docker: &dtos.SingularityDockerInfo{},
					},
					Resources: &dtos.Resources{},
				},
			}, nil)

			whip[url] = co
			return cl
		},
	}

	res, err := dep.GetRunningDeployment(map[string]string{`test`: `http://test-singularity.org/`})
	assert.NoError(err)
	assert.NotNil(res)
}
