package singularity

import "github.com/opentable/go-singularity/dtos"

// DeployReader encapsulates the methods required to read Singularity
// requests and deployments.
//
// DeployReader is satisfied by *singularity.Client.
type DeployReader interface {
	GetRequests() (dtos.SingularityRequestParentList, error)
	GetRequest(requestID string) (*dtos.SingularityRequestParent, error)
	GetDeploy(requestID, deployID string) (*dtos.SingularityDeployHistory, error)
	GetDeploys(requestID string, count, page int32) (dtos.SingularityDeployHistoryList, error)
}
