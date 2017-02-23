package singularity

import (
	"fmt"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/coaxer"
	"github.com/pkg/errors"
)

// DeploymentBuilder is responsible for constructing a sous.Deployment from a
// Singularity deployment.
type DeploymentBuilder struct {
	*requestContext
	promise coaxer.Promise
	// DeployID is the singularity deploy ID
	// (not to be confused with sous.DeployID).
	// You can always expect DeployID to have a meaningful value.
	DeployID string
}

func newDeploymentBuilder(rc *requestContext, deployID string) *DeploymentBuilder {

	promise := c.Coax(rc.Context, func() (interface{}, error) {
		return maybeRetryable(rc.Client.GetDeploy(rc.RequestID, deployID))
	}, "get deployment %q", deployID)

	return &DeploymentBuilder{
		requestContext: rc,
		DeployID:       deployID,
		promise:        promise,
	}
}

// Errorf returns a formatted error with contextual info about which Singularity
// deploy the error relates to.
func (db *DeploymentBuilder) Errorf(format string, a ...interface{}) error {
	prefix := fmt.Sprintf("singularity deployment %q (request %q)", db.DeployID, db.RequestID)
	message := fmt.Sprintf(format, a...)
	return fmt.Errorf("%s: %s", prefix, message)
}

// Deployment returns the Deployment.
func (db *DeploymentBuilder) Deployment() (*sous.Deployment, sous.DeployStatus, error) {
	if err := db.promise.Err(); err != nil {
		return nil, sous.DeployStatusUnknown, err
	}
	deployHistoryItem := db.promise.Value().(*dtos.SingularityDeployHistory)

	deploy := deployHistoryItem.Deploy

	if deploy.ContainerInfo == nil ||
		deploy.ContainerInfo.Docker == nil ||
		deploy.ContainerInfo.Docker.Image == "" {
		return nil, sous.DeployStatusUnknown, db.Errorf("no docker image specified at deploy.ContainerInfo.Docker.Image")
	}
	dockerImage := deploy.ContainerInfo.Docker.Image

	// This is our only dependency on the registry.
	labels, err := db.Registry.ImageLabels(dockerImage)
	sourceID, err := docker.SourceIDFromLabels(labels)
	if err != nil {
		return nil, sous.DeployStatusUnknown, errors.Wrapf(err, "getting source ID")
	}

	requestParent, err := db.RequestParent()
	if err != nil {
		return nil, sous.DeployStatusUnknown, err
	}
	if requestParent.Request == nil {
		return nil, sous.DeployStatusUnknown, db.Errorf("requestParent contains no request")
	}

	return mapDeployHistoryToDeployment(db.Cluster, sourceID, requestParent.Request, deployHistoryItem)
}
