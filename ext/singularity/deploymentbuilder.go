package singularity

import (
	"fmt"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/util/coaxer"
)

// DeployHistoryBuilder is responsible for constructing a sous.Deployment from a
// Singularity deployment.
type DeployHistoryBuilder struct {
	*requestContext
	promise coaxer.Promise
	// DeployID is the singularity deploy ID
	// (not to be confused with sous.DeployID).
	// You can always expect DeployID to have a meaningful value.
	DeployID string
}

func newDeploymentHistoryBuilder(rc *requestContext, deployID string) *DeployHistoryBuilder {
	promise := c.Coax(rc.Context, func() (interface{}, error) {
		return maybeRetryable(rc.Client.GetDeploy(rc.RequestID, deployID))
	}, "get deployment %q from request %q", deployID, rc.RequestID)

	return &DeployHistoryBuilder{
		requestContext: rc,
		DeployID:       deployID,
		promise:        promise,
	}
}

// DeployHistory returns a single singularity deploy history entry, or an error.
func (db *DeployHistoryBuilder) DeployHistory() (*dtos.SingularityDeployHistory, error) {
	if err := db.promise.Err(); err != nil {
		return nil, err
	}
	return db.promise.Value().(*dtos.SingularityDeployHistory), nil
}

// Errorf returns a formatted error with contextual info about which Singularity
// deploy the error relates to.
func (db *DeployHistoryBuilder) Errorf(format string, a ...interface{}) error {
	prefix := fmt.Sprintf("singularity deployment %q (request %q)", db.DeployID, db.RequestID)
	message := fmt.Sprintf(format, a...)
	return fmt.Errorf("%s: %s", prefix, message)
}
