package sous

//go:generate ggen cmap.CMap(cmap.go) sous.DeployStates(deploystates.go) CMKey:DeployID Value:*DeployState

// A DeployState represents the state of a deployment in an external cluster.
// It wraps Deployment and adds Status.
type DeployState struct {
	Deployment Deployment
	Status     DeployStatus
	// FailedDeployment is populated with the latest attempted deployment, if it
	// failed.
	FailedDeployment *Deployment
	// FailedDeploymentReason is a human-readable string explaining why
	// FailedDeployment failed.
	FailedDeploymentReason string
}

// DeployStatus represents the status of a deployment in an external cluster.
type DeployStatus int

const (
	// DeployStatusUnknown represents any deployment status that is
	// unrecognised.
	DeployStatusUnknown DeployStatus = iota
	// DeployStatusAny represents any deployment status.
	DeployStatusAny
	// DeployStatusPending means the deployment has been requested in the
	// cluster, but is not yet running.
	DeployStatusPending
	// DeployStatusActive means the deployment is up and running.
	DeployStatusActive
	// DeployStatusFailed means the deployment is broken due to either broken
	// code or configuration. This needs to be fixed by the deployment's owner
	// e.g. by fixing the code, or by changing the configuration. It will not
	// be re-tried.
	DeployStatusFailed
	// DeployStatusNotEnoughResources means the deployment has failed because
	// the cluster does not have enough resources to schedule all the tasks.
	DeployStatusNotEnoughResources
	// DeployStatusCancelled means a user cancelled the deployment.
	DeployStatusCancelled
	// DeployStatusPendingDeployRemoved
	//DeployStatusPendingDeployRemoved
	// DeployStatusLoadBalancerUpdateFailed
	//DeployStatusLoadBalancerUpdateFailed
	// DeployStatusTaskNeverEnteredRunning
	//DeployStatusTaskNeverEnteredRunning
)

func (ds DeployState) String() string {
	return ds.Deployment.String()
}

// ID returns the DeployID of this DeployState.
func (ds DeployState) ID() DeployID {
	return ds.Deployment.ID()
}

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
