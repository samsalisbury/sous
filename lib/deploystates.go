package sous

import "fmt"

// Diff2 calculates the differences between two DeployStates.
func (ds DeployStates) Diff2(other DeployStates) (bool, []string) {
	var diffs []string
	diff := func(format string, a ...interface{}) { diffs = append(diffs, fmt.Sprintf(format, a...)) }

	d := ds.Snapshot()
	o := other.Snapshot()

	if len(d) != len(o) {
		diff("number of deploy states; this: %d, other: %d", len(d), len(o))
	}

	for did, thisDeployState := range d {
		otherDeployState, ok := o[did]
		if !ok {
			diff("other missing deploy state %q", did)
			continue
		}
		dsDifferent, dsDiffs := thisDeployState.Diff(otherDeployState)
		if dsDifferent {
			diff("differences in deploy state %q", did)
			for _, dsDiff := range dsDiffs {
				diff("in deploy state %q: %s", did, dsDiff)
			}
		}
	}

	for did := range o {
		if _, ok := d[did]; !ok {
			diff("other has extra deploy state %q", did)
		}
	}

	return len(diffs) != 0, diffs
}

// ToDeployStatesWithStatus returns a DeployStates containing all these
// Deployments wrapped in DeployStatuses with Status == status.
func (ds Deployments) ToDeployStatesWithStatus(status DeployStatus) DeployStates {
	deployStates := NewDeployStates()
	for did, d := range ds.Snapshot() {
		deployStates.Set(did, &DeployState{Deployment: *d, Status: status})
	}
	return deployStates
}
