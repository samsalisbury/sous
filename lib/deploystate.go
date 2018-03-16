package sous

import "fmt"

//go:generate ggen cmap.CMap(cmap.go) sous.DeployStates(deploystates.go) CMKey:DeployID Value:*DeployState
//go:generate stringer -type=DeployStatus

// A DeployState represents the state of a deployment in an external cluster.
// It wraps Deployment and adds Status.
type DeployState struct {
	Deployment
	Status          DeployStatus
	ExecutorMessage string
	ExecutorData    interface{}
}

// DeployStatus represents the status of a deployment in an external cluster.
type DeployStatus int

const (
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

func (ds DeployState) String() string {
	return fmt.Sprintf("DEPLOYMENT:%s STATUS:%s EXECUTORDATA:%v", ds.Deployment.String(), ds.Status, ds.ExecutorData)
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

// Final reports whether we should expect this DeployState to be finished -
// in other words, DeployState.Final() -> false implies that a subsequent
// DeployState will have a different status; polling components will want to poll again.
func (ds DeployState) Final() bool {
	switch ds.Status {
	default:
		return false
	case DeployStatusActive, DeployStatusFailed:
		return true
	}
}

// Diff computes the list of differences between two DeployStates and returns
// "true" if they're different, along with a list of differences
func (ds *DeployState) Diff(o *DeployState) (bool, []string) {
	// XXX uses deployment.Diff
	_, depS := ds.Deployment.Diff(&o.Deployment)

	if ds.Status != o.Status {
		depS = append(depS, fmt.Sprintf("status: this: %s other: %s", ds.Status, o.Status))
	}
	return len(depS) > 0, depS
}
