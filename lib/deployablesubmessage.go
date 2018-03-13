package sous

import (
	"github.com/opentable/sous/util/logging"
)

type deployableSubmessage struct {
	deployable    *Deployable
	deploymentSub logging.EachFielder
	prefix        string
}

// NewDeployableSubmessage creates a new EachFielder that produces fields for a Deployable..
func NewDeployableSubmessage(prefix string, dep *Deployable) logging.EachFielder {
	smsg := &deployableSubmessage{
		prefix:     prefix,
		deployable: dep,
	}

	if dep != nil {
		smsg.deploymentSub = NewDeploymentSubmessage(prefix, dep.Deployment)
	} else {
		smsg.deploymentSub = NewDeploymentSubmessage(prefix, nil)
	}

	return smsg
}

func (msg *deployableSubmessage) buildArtifactFields(f logging.FieldReportFn) {
	ba := msg.deployable.BuildArtifact

	if ba == nil {
		return
	}

	f(msg.prefix+"-artifact-name", ba.Name)
	f(msg.prefix+"-artifact-type", ba.Type)
	f(msg.prefix+"-artifact-qualities", ba.Qualities.String())
}

// EachField implements EachFielder on deployableSubmessage.
func (msg *deployableSubmessage) EachField(f logging.FieldReportFn) {
	d := msg.deployable
	if d == nil {
		return
	}
	f(msg.prefix+"-status", d.Status.String())

	msg.buildArtifactFields(f)
	msg.deploymentSub.EachField(f)
}
