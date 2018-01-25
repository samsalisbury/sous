package sous

import (
	"fmt"

	"github.com/opentable/sous/util/logging"
)

// A DeployablePairSubmessage collects the common bits of logging events with
// a DeployablePair in their context. e.g. rectifier differences and deployments.
type DeployablePairSubmessage struct {
	pair     *DeployablePair
	priorSub *DeployableSubmessage
	postSub  *DeployableSubmessage
}

// NewDeployablePairSubmessage returns a new DeployablePairSubmessage.
func NewDeployablePairSubmessage(pair *DeployablePair) *DeployablePairSubmessage {
	msg := &DeployablePairSubmessage{
		pair: pair,
	}

	if pair != nil {
		msg.priorSub = NewDeployableSubmessage("sous-prior", pair.Prior)
		msg.postSub = NewDeployableSubmessage("sous-post", pair.Post)
	}

	return msg
}

// EachField implements the EachFielder interface on DeployablePairSubmessage.
func (msg *DeployablePairSubmessage) EachField(f logging.FieldReportFn) {
	if msg.pair == nil {
		return
	}

	f("sous-deployment-id", msg.pair.ID().String())
	f("sous-manifest-id", msg.pair.ID().ManifestID.String())
	f("sous-diff-disposition", msg.pair.Kind().String())
	if msg.pair.Kind() == ModifiedKind {
		f("sous-deployment-diffs", msg.pair.Diffs().String())
	} else {
		f("sous-deployment-diffs", fmt.Sprintf("No detailed diff because pairwise diff kind is %q", msg.pair.Kind()))
	}

	msg.priorSub.EachField(f)
	msg.postSub.EachField(f)
}
