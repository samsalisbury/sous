package sous

import "github.com/nyarly/spies"

type (
	// Deployer describes a complete deployment system, which is able to create,
	// read, update, and delete deployments.
	Deployer interface {
		RunningDeployments(reg Registry, from Clusters) (DeployStates, error)
		Rectify(*DeployablePair) DiffResolution
		Status(Registry, Clusters, *DeployablePair) (DeployState, error)
	}

	// DeployerSpy is a noop deployer.
	DeployerSpy struct {
		*spies.Spy
	}
)

// NewDummyDeployer creates a DummyDeployer
func NewDummyDeployer() Deployer {
	d, c := NewDeployerSpy()
	c.MatchMethod("RunningDeployments", spies.AnyArgs, NewDeployStates(), nil)
	c.MatchMethod("Rectify", spies.AnyArgs, DiffResolution{})
	c.MatchMethod("Status", spies.AnyArgs, DeployState{}, nil)
	return d
}

func NewDeployerSpy() (Deployer, *spies.Spy) {
	spy := spies.NewSpy()

	return &DeployerSpy{Spy: spy}, spy
}

// RunningDeployments implements Deployer
func (dd *DeployerSpy) RunningDeployments(reg Registry, from Clusters) (DeployStates, error) {
	res := dd.Called(reg, from)
	return res.Get(0).(DeployStates), res.Error(1)
}

// Rectify implements Deployer
func (dd *DeployerSpy) Rectify(p *DeployablePair) DiffResolution {
	res := dd.Called(p)
	return res.Get(0).(DiffResolution)
}

// Status implements Deployer
func (dd *DeployerSpy) Status(r Registry, c Clusters, p *DeployablePair) (DeployState, error) {
	res := dd.Called(r, c, p)
	return res.Get(0).(DeployState), res.Error(1)
}
