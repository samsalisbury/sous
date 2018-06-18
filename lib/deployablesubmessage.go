package sous

import (
	"github.com/opentable/sous/util/logging"
)

type deployableSubmessage struct {
	deployable    *Deployable
	deploymentSub logging.EachFielder
	fields        map[string]logging.FieldName
}

// NewDeployableSubmessage creates a new EachFielder that produces fields for a Deployable..
func NewDeployableSubmessage(prefix string, dep *Deployable) logging.EachFielder {
	smsg := &deployableSubmessage{
		deployable: dep,
	}

	if dep != nil {
		smsg.deploymentSub = NewDeploymentSubmessage(prefix, dep.Deployment)
	} else {
		smsg.deploymentSub = NewDeploymentSubmessage(prefix, nil)
	}

	switch prefix {
	default:
		smsg.fields = map[string]logging.FieldName{
			"artifact-name":      "unknown-artifact-name",
			"artifact-type":      "unknown-artifact-type",
			"artifact-qualities": "unknown-artifact-qualities",
			"status":             "unknown-status",
		}
	case "sous-prior":
		smsg.fields = map[string]logging.FieldName{
			"artifact-name":      logging.SousPriorArtifactName,
			"artifact-type":      logging.SousPriorArtifactType,
			"artifact-qualities": logging.SousPriorArtifactQualities,
			"status":             logging.SousPriorStatus,
		}
	case "sous-post":
		smsg.fields = map[string]logging.FieldName{
			"artifact-name":      logging.SousPostArtifactName,
			"artifact-type":      logging.SousPostArtifactType,
			"artifact-qualities": logging.SousPostArtifactQualities,
			"status":             logging.SousPostStatus,
		}
	}

	return smsg
}

func (msg *deployableSubmessage) buildArtifactFields(f logging.FieldReportFn) {
	ba := msg.deployable.BuildArtifact

	if ba == nil {
		return
	}

	ba.EachField(func(n logging.FieldName, v interface{}) {
		f(msg.fields[string(n)], v)
	})
}

// EachField implements EachFielder on deployableSubmessage.
func (msg *deployableSubmessage) EachField(f logging.FieldReportFn) {
	d := msg.deployable
	if d == nil {
		return
	}
	f(msg.fields["status"], d.Status.String())

	msg.buildArtifactFields(f)
	msg.deploymentSub.EachField(f)
}
