package sous

type (
	// Deployer describes a complete deployment system, which is able to create,
	// read, update, and delete deployments.
	Deployer interface {
		RunningDeployments(reg Registry, from Clusters) (Deployments, error)
		RectifyCreates(<-chan *Deployable, chan<- DiffResolution)
		RectifyDeletes(<-chan *Deployable, chan<- DiffResolution)
		RectifyModifies(<-chan *DeployablePair, chan<- DiffResolution)
	}

	// DummyDeployer is a noop deployer.
	DummyDeployer struct {
		deps Deployments
	}
)

// NewDummyDeployer creates a DummyDeployer
func NewDummyDeployer() *DummyDeployer {
	return &DummyDeployer{deps: NewDeployments()}
}

// RunningDeployments implements Deployer
func (dd *DummyDeployer) RunningDeployments(reg Registry, from Clusters) (Deployments, error) {
	return dd.deps, nil
}

// RectifyCreates implements Deployer
func (dd *DummyDeployer) RectifyCreates(<-chan *Deployable, chan<- DiffResolution) {}

// RectifyDeletes implements Deployer
func (dd *DummyDeployer) RectifyDeletes(<-chan *Deployable, chan<- DiffResolution) {}

// RectifyModifies implements Deployer
func (dd *DummyDeployer) RectifyModifies(<-chan *DeployablePair, chan<- DiffResolution) {}
