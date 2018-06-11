package sous

import "github.com/opentable/sous/util/logging"

// A Deployable is the pairing of a Deployment and the resolved image that can
// (or has) be used to deploy it.
type Deployable struct {
	Status DeployStatus
	*Deployment
	*BuildArtifact
}

// EachField ... you get the idea by now
func (d Deployable) EachField(f logging.FieldReportFn) {
	f(logging.DeployStatus, d.Status)
	if d.Deployment != nil {
		d.Deployment.EachField(f)
	}

	if d.BuildArtifact != nil {
		d.BuildArtifact.EachField(f)
	}
}
