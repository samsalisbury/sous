package singularity

import (
	"fmt"
	"log"

	"github.com/opentable/go-singularity/dtos"
)

type testDeployReader struct {
	Fixture *testFixture
}

func (tdr *testDeployReader) GetTestRequest(requestID string) (*testRequestParent, error) {
	did, err := ParseRequestID(requestID)
	if err != nil {
		log.Panic(err)
	}
	// Let these panic if there is nothing there.
	cluster, ok := tdr.Fixture.Clusters[did.Cluster]
	if !ok {
		log.Panicf("No cluster called %q", did.Cluster)
	}
	baseURL := cluster.BaseURL
	singularity, ok := tdr.Fixture.Singularities[baseURL]
	if !ok {
		log.Panicf("No Singularity for base URL %q (of cluster %q)", baseURL, did.Cluster)
	}
	request, ok := singularity.Requests[requestID]
	if !ok {
		return nil, fmt.Errorf("no request named %q in the fixture", requestID)
	}
	return request, nil
}

// GetRequests implements DeployReader.GetRequests.
func (tdr *testDeployReader) GetRequests() (dtos.SingularityRequestParentList, error) {
	rpl := dtos.SingularityRequestParentList{}
	for _, singularity := range tdr.Fixture.Singularities {
		for _, request := range singularity.Requests {
			if request.Error != nil {
				return nil, request.Error
			}
			rpl = append(rpl, request.RequestParent)
		}
	}
	return rpl, nil
}

// GetRequest implements DeployReader.GetRequest.
func (tdr *testDeployReader) GetRequest(requestID string) (*dtos.SingularityRequestParent, error) {
	request, err := tdr.GetTestRequest(requestID)
	if err != nil {
		return nil, httpErr(404, err.Error())
	}
	if request.RequestParent == nil {
		log.Panicf("testRequest has no RequestParent")
	}
	return request.RequestParent, nil
}

// GetDeploy implements DeployReader.GetDeploy.
func (tdr *testDeployReader) GetDeploy(requestID, deployID string) (*dtos.SingularityDeployHistory, error) {
	if deployID == "" {
		log.Panic("GetDeploy passed an empty deployID")
	}
	request, err := tdr.GetTestRequest(requestID)
	if err != nil {
		// TODO: Find out what Swaggering does and ensure we are emulating that.
		return nil, httpErr(404, "no deploy %q; no request named %q in the fixture", deployID, requestID)
	}
	deploy, ok := request.Deploys[deployID]
	if !ok {
		return nil, httpErr(404, "no deploy %q in request %q", deployID, requestID)
	}
	return deploy.DeployHistoryItem, nil
}
