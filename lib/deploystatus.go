package sous

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

// Failed reports whether a DeployStatus represents a failed state.
func (ds DeployStatus) Failed() bool {
	switch ds {
	default:
		return false
	case DeployStatusFailed:
		return true
	}

}
