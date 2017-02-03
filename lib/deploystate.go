package sous

//go:generate ggen cmap.CMap(cmap.go) sous.DeployStates(deploystates.go) CMKey:DeployID Value:*DeployState

// A DeployState represents the state of a deployment in an external cluster.
// It wraps Deployment and adds Status.
type DeployState struct {
	*Deployment
	Status DeployStatus
}

// DeployStatus represents the status of a deployment in an external cluster.
type DeployStatus string

const (
	// DeployStatusPending means the deployment has been requested in the
	// cluster, but is not yet running.
	DeployStatusPending DeployStatus = "pending"
	// DeployStatusActive means the deployment is up and running.
	DeployStatusActive = "active"
	// DeployStatusFailed means the deployment has failed.
	DeployStatusFailed = "failed"
)

func (ds DeployState) Clone() *DeployState {
	ds.Deployment = ds.Deployment.Clone()
	return &ds
}
