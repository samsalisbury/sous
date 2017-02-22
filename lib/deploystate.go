package sous

import "fmt"

//go:generate stringer -type=DeployStatus
//go:generate ggen cmap.CMap(cmap.go) sous.DeployStates(deploystates.go) CMKey:DeployID Value:*DeployState

// A DeployState represents the state of a deployment in an external cluster.
type DeployState struct {
	// Status is the overall status of this DeployState.
	// It is equal to LastAttemptedDeployStatus.
	Status DeployStatus

	Deployment Deployment

	//// ActiveDeployment is the deployment that is currently running or pending.
	//ActiveDeployment Deployment
	//// ActiveDeployStatus is the status of ActiveDeployment. It is either
	//// DeployStatusSucceeded or DeployStatusPending.
	//ActiveDeployStatus DeployStatus
	//// LastAttemptedDeployment is the last deployment that was attempted.
	//// This may be the same as ActiveDeployment if ActiveDeployment was
	//// successful.
	//LastAttemptedDeployment *Deployment
	//// LastAttemptedDeployStatus is the status of LastAttemptedDeployment.
	//// It can have any DeployStatus value.
	//LastAttemptedDeployStatus DeployStatus
}

func (ds *DeployState) String() string {
	return ds.Deployment.String()
}

// ID returns the DeployID.
func (ds *DeployState) ID() DeployID {
	return ds.Deployment.ID()
}

// Tabbed returns the active deployment in human-readable form.
func (ds *DeployState) Tabbed() string {
	return ds.Deployment.Tabbed()
}

// Diff returns true, list of diffs if o != ds. Otherwise returns false, nil.
func (ds *DeployState) Diff(o *DeployState) (bool, []string) {
	_, diffs := ds.Deployment.Diff(&o.Deployment)
	//	if o.ActiveDeployStatus != ds.ActiveDeployStatus {
	//		diffs = append(diffs, fmt.Sprintf("ActiveDeployStatus; this: %s, other: %s",
	//			ds.ActiveDeployStatus, o.ActiveDeployStatus))
	//	}
	//	if o.LastAttemptedDeployStatus != ds.LastAttemptedDeployStatus {
	//		diffs = append(diffs, fmt.Sprintf("LastAttemptedDeployStatus; this: %s, other: %s",
	//			ds.LastAttemptedDeployStatus, o.LastAttemptedDeployStatus))
	//	}
	if o.Status != ds.Status {
		diffs = append(diffs, fmt.Sprintf("Status; this: %s, other: %s",
			ds.Status, o.Status))
	}
	return len(diffs) != 0, diffs
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
	DeployStatusAny
	// DeployStatusPending means the deployment has been requested in the
	// cluster, but is not yet running.
	DeployStatusPending
	// DeployStatusActive means the deployment is up and running.
	DeployStatusActive
	// DeployStatusFailed means the deployment has failed.
	DeployStatusFailed
	// DeployStatusSucceeded means the deployment succeeded at the time, but
	// does not tell us about whether it is currently running or not.
	// Note: This is different in nature than some of the other statuses
	// here, because it only speaks about the immediate results of a deployment.
	// We may need a new type for these kinds of values eventually.
	DeployStatusSucceeded
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
