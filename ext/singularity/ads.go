package singularity

import (
	"fmt"
	"sync"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
)

// Deployer implements sous.Deployer for a single sous Cluster running on
// Singularity.
type Deployer struct {
	Client  *singularity.Client
	Cluster sous.Cluster
}

// adsBuild represents the building of a single sous.DeployStates from a
// single Singularity-hosted cluster.
type adsBuild struct {
	Client  *singularity.Client
	Cluster sous.Cluster
	Errors  chan error
}

// requestContext is a mixin used for DeploymentBuilder and DeployStateBuilder.
type requestContext struct {
	// Client to be used for all requests to Singularity.
	Client *singularity.Client
	// The Cluster this DeployStateBuilder is working on.
	// Note that one Singularity cluster may host multiple Sous clusters.
	Cluster *sous.Cluster
	// RequestID is the singularity request ID this builder is working on.
	// This field is populated by NewDeployStateBuilder, and you can always
	// assume that it is populated with a meaningful value.
	RequestID string
}

// DeployStateBuilder visits each phase in the life-cycle of building a
// deployment and gathers the data needed to populate its Result field.
type DeployStateBuilder struct {
	requestContext
	// CurrentDeployID is the singularity deploy ID of the currently active or
	// pending deployment for request RequestID. It may be empty if there is no
	// active or pending deployment.
	CurrentDeployID string
	// CurrentDeployStatus is the status of the current singularity deployment,
	// either sous.DeployStatusActive or sous.DeployStatusComing.
	CurrentDeployStatus sous.DeployStatus
}

// DeploymentBuilder is responsible for constructing a sous.Deployment from a
// Singularity deployment.
type DeploymentBuilder struct {
	sync.WaitGroup
	requestContext
	// DeployID is the singularity deploy ID
	// (not to be confused with sous.DeployID).
	// You can always expect DeployID to have a meaningful value.
	DeployID string
}

func newADSBuild(client *singularity.Client, cluster sous.Cluster) *adsBuild {
	return &adsBuild{
		Client:  client,
		Cluster: cluster,
		Errors:  make(chan error),
	}
}

// RunningDeployments uses a new adsBuild to construct sous deploy states.
func (d *Deployer) RunningDeployments(reg sous.Registry, _ sous.Clusters) (*sous.DeployStates, error) {
	return newADSBuild(d.Client, d.Cluster).DeployStates()
}

// DeployStates returns all deploy states.
func (ab *adsBuild) DeployStates() (*sous.DeployStates, error) {

	// Grab the list of all requests from Singularity.
	requests, err := ab.Client.GetRequests()
	if err != nil {
		return nil, err
	}

	deployStates := sous.NewDeployStates()
	var wg sync.WaitGroup

	// Start gathering all request concurrently.
	for _, r := range requests {
		wg.Add(1)
		request := r
		go func() {
			defer wg.Done()
			dsb := ab.newDeployStateBuilder(request)
			ds, err := dsb.DeployState()
			if err != nil {
				ab.Errors <- err
				return
			}
			deployStates.Add(ds)
		}()
	}

	wg.Wait()

	return &deployStates, nil
}

func (ab *adsBuild) Errorf(format string, a ...interface{}) error {
	prefix := fmt.Sprintf("reading from cluster %q", ab.Cluster.Name)
	message := fmt.Sprintf(format, a...)
	return fmt.Errorf("%s: %s", prefix, message)
}

func (ab *adsBuild) newDeployStateBuilder(rp *dtos.SingularityRequestParent) *DeployStateBuilder {
	deployID, status := getCurrentDeployIDAndStatus(rp.RequestDeployState)
	return &DeployStateBuilder{
		requestContext: requestContext{
			Client:    ab.Client,
			Cluster:   &ab.Cluster,
			RequestID: rp.Request.Id,
		},
		CurrentDeployID:     deployID,
		CurrentDeployStatus: status,
	}
}

func getCurrentDeployIDAndStatus(rds *dtos.SingularityRequestDeployState) (string, sous.DeployStatus) {
	// If there is a pending request, that's the one we care about from Sous'
	// point of view, so preferentially return that.
	if pending := rds.PendingDeploy; pending != nil {
		return pending.DeployId, sous.DeployStatusPending
	}
	// If there's nothing pending, let's try to return the active deploy.
	if active := rds.ActiveDeploy; active != nil {
		return active.DeployId, sous.DeployStatusActive
	}
	return "", sous.DeployStatusNotRunning
}

// DeployState returns the Sous deploy state.
func (ds *DeployStateBuilder) DeployState() (*sous.DeployState, error) {

	var current, previous sous.Deployment
	var currentStatus, previousStatus sous.DeployStatus

	currentDeployBuilder := ds.newDeploymentBuilder(ds.CurrentDeployID)
	previousDeployBuilder := ds.newDeploymentBuilder("TODO GET LAST DEPLOY ID")

	currentDeployBuilder.Fetch(&current, &currentStatus)
	previousDeployBuilder.Fetch(&previous, &previousStatus)

	currentDeployBuilder.Wait()
	previousDeployBuilder.Wait()

	return &sous.DeployState{
		Deployment: current,
		Status:     currentStatus,
	}, nil
}

func (ds *DeployStateBuilder) newDeploymentBuilder(deployID string) *DeploymentBuilder {
	return &DeploymentBuilder{
		requestContext: ds.requestContext,
		DeployID:       deployID,
	}
}

// Fetch populates d and s by making HTTP requests to Singularity.
func (db *DeploymentBuilder) Fetch(d *sous.Deployment, s *sous.DeployStatus) error {
	db.Add(1)
	defer db.Done()
	return nil
}
