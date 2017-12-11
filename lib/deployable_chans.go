package sous

import (
	"sync"
)

type (
	// A DeployableChans is a bundle of channels describing actions to take on a
	// cluster
	DeployableChans struct {
		Pairs chan *DeployablePair
		Errs  chan *DiffResolution
		sync.WaitGroup
	}

	// A DeployablePair is a pair of deployables, describing a "before and after"
	// situation, where the Prior Deployable is the known state and the Post
	// Deployable is the desired state.
	DeployablePair struct {
		Kind         DeployablePairKind
		Diffs        Differences
		Prior, Post  *Deployable
		name         DeploymentID
		ExecutorData interface{}
	}

	// DeployablePairKind describes the disposition of a DeployablePair
	DeployablePairKind int

	diffSet struct {
		Pairs DeployablePairs
	}
)

const (
	// SameKind prior is unchanged from post
	SameKind DeployablePairKind = iota
	// AddedKind means an added deployable - there's no prior
	AddedKind
	// RemovedKind means a removed deployable - no post
	RemovedKind
	// ModifiedKind means modified deployable - post and prior are different
	ModifiedKind
)

// NewDeployableChans returns a new DeployableChans with channels buffered to
// size.
func NewDeployableChans(size ...int) *DeployableChans {
	var s int
	if len(size) > 0 {
		s = size[0]
	}
	return &DeployableChans{
		Pairs: make(chan *DeployablePair, s),
		Errs:  make(chan *DiffResolution, s),
	}
}

// Close closes all the channels in a DeployableChans in a single action
func (d *DeployableChans) Close() {
	close(d.Pairs)
	close(d.Errs)
}

func (d *DeployableChans) collect() diffSet {
	ds := newDiffSet()

	for m := range d.Pairs {
		ds.Pairs = append(ds.Pairs, m)
	}
	return ds
}

// ID returns the ID of this DeployablePair.
func (dp *DeployablePair) ID() DeploymentID {
	return dp.name
}

func (kind DeployablePairKind) String() string {
	switch kind {
	default:
		return "unknown"
	case SameKind:
		return "same"
	case AddedKind:
		return "added"
	case RemovedKind:
		return "removed"
	case ModifiedKind:
		return "modified"
	}
}

func (kind DeployablePairKind) ResolveVerb() string {
	switch kind {
	default:
		return ""
	case SameKind:
		return "take no action"
	case AddedKind:
		return "create new deployment"
	case RemovedKind:
		return "delete existing deployment"
	case ModifiedKind:
		return "update existing deployment"
	}
}
