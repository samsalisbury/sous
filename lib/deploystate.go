package sous

//go:generate ggen cmap.CMap(cmap.go) sous.DeployStates(deploystates.go) CMKey:DeployID Value:*DeployState

// A DeployState represents the state of a deployment in an external cluster.
// It wraps Deployment and adds Status.
type DeployState struct {
	// Active is the currently active, or pending deployment.
	// We include pending with active, since Sous should wait for
	// a pending deployment to either fail or succeed before considering
	// it for rectification of changes.
	Active Deployment
	// Status is the deploy status of the active deployment, either
	// DeployStatusPending or DeployStatusActive.
	ActiveStatus DeployStatus
	// ActiveArtifact is the artifact currently running.
	ActiveArtifact BuildArtifact
	// Failed is populated with the latest attempted deployment, if it
	// failed.
	Failed *Deployment
	// FailedReason is a human-readable string explaining why
	// FailedDeployment failed. It is empty when Failed is nil.
	FailedReason string
	// FailedStatus is the status of Failed.
	FailedStatus DeployStatus
	// FailedArtifact is the artifact deployed in the failed deployment.
	FailedArtifact BuildArtifact
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
	return ds.Active.String()
}

// ID returns the DeployID of this DeployState.
func (ds DeployState) ID() DeployID {
	return ds.Active.ID()
}

// Clone returns an independent clone of this DeployState.
func (ds DeployState) Clone() *DeployState {
	ds.Active = *ds.Active.Clone()
	return &ds
}

// IgnoringStatus returns a Deployments containing all the nested deployments
// in this DeployStates.
func (ds DeployStates) IgnoringStatus() Deployments {
	deployments := NewDeployments()
	for key, value := range ds.Snapshot() {
		deployments.Set(key, &value.Active)
	}
	return deployments
}
