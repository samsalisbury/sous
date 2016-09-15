package sous

type (
	// Deployer describes a complete deployment system, which is able to create,
	// read, update, and delete deployments.
	Deployer interface {
		RunningDeployments(from Clusters) (Deployments, error)
		RectifyCreates(<-chan *Deployment, chan<- RectificationError)
		RectifyDeletes(<-chan *Deployment, chan<- RectificationError)
		RectifyModifies(<-chan *DeploymentPair, chan<- RectificationError)
	}

	DummyDeployer struct {
		deps Deployments
	}
)

// NewDummyDeployer creates a DummyDeployer
func NewDummyDeployer() *DummyDeployer {
	return &DummyDeployer{deps: NewDeployments()}
}

// RunningDeployments implements Deployer
func (dd *DummyDeployer) RunningDeployments(from Clusters) (Deployments, error) {
	return dd.deps, nil
}

// RectifyCreates implements Deployer
func (dd *DummyDeployer) RectifyCreates(<-chan *Deployment, chan<- RectificationError) {}

// RectifyDeletes implements Deployer
func (dd *DummyDeployer) RectifyDeletes(<-chan *Deployment, chan<- RectificationError) {}

// RectifyModifies implements Deployer
func (dd *DummyDeployer) RectifyModifies(<-chan *DeploymentPair, chan<- RectificationError) {}
