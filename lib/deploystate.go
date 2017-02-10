package sous

//go:generate ggen cmap.CMap(cmap.go) sous.DeployStates(deploystates.go) CMKey:DeployID Value:*DeployState

// A DeployState represents the state of a deployment in an external cluster.
// It wraps Deployment and adds Status.
type DeployState struct {
	Deployment
	Status DeployStatus
}

// DeployStatus represents the status of a deployment in an external cluster.
type DeployStatus int

const (
	// InvalidDeployStatus is an invalid value in all contexts, it is the
	// zero DeployStatus.
	InvalidDeployStatus DeployStatus = iota
	// DeployStatusUnknown means no deployment status has been determined.
	DeployStatusUnknown
	// DeployStatusNotRunning means there is no running deployment.
	DeployStatusNotRunning
	// DeployStatusAny represents any deployment status.
	DeployStatusAny DeployStatus = iota
	// DeployStatusPending means the deployment has been requested in the
	// cluster, but is not yet running.
	DeployStatusPending
	// DeployStatusActive means the deployment is up and running.
	DeployStatusActive
	// DeployStatusFailed means the deployment has failed.
	DeployStatusFailed
)

// Clone returns an independent clone of this DeployState.
func (ds DeployState) Clone() *DeployState {
	ds.Deployment = *ds.Deployment.Clone()
	return &ds
}

// IgnoringStatus returns a Deployments containing all the nested deployments
// in this DeployStates.
func (ds DeployStates) IgnoringStatus() Deployments {
	deployments := NewDeployments()
	for key, value := range ds.Snapshot() {
		deployments.Set(key, &value.Deployment)
	}
	return deployments
}
