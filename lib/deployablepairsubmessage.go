package sous

import (
	"fmt"

	"github.com/opentable/sous/util/logging"
)

// A deployablePairSubmessage collects the common bits of logging events with
// a DeployablePair in their context. e.g. rectifier differences and deployments.
type deployablePairSubmessage struct {
	pair     *DeployablePair
	priorSub logging.EachFielder
	postSub  logging.EachFielder
}

// NewDeployablePairSubmessage returns a new deployablePairSubmessage.
func NewDeployablePairSubmessage(pair *DeployablePair) logging.Submessage {
	msg := &deployablePairSubmessage{
		pair: pair,
	}

	if pair != nil {
		msg.priorSub = NewDeployableSubmessage("sous-prior", pair.Prior)
		msg.postSub = NewDeployableSubmessage("sous-post", pair.Post)
	}

	return msg
}

func (msg *deployablePairSubmessage) RecommendedLevel() logging.Level {
	if msg.pair.Post == nil {
		return logging.WarningLevel
	}

	if msg.pair.Prior == nil {
		return logging.InformationLevel
	}

	if len(msg.pair.Diffs()) == 0 {
		return logging.DebugLevel
	}

	return logging.InformationLevel
}

// EachField implements the EachFielder interface on deployablePairSubmessage.
func (msg *deployablePairSubmessage) EachField(f logging.FieldReportFn) {
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

	if msg.priorSub != nil {
		msg.priorSub.EachField(f)
	}
	if msg.priorSub != nil {
		msg.postSub.EachField(f)
	}
}
