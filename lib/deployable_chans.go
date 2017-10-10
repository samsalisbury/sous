package sous

import (
	"sync"
)

type (
	// A DeployableChans is a bundle of channels describing actions to take on a
	// cluster
	DeployableChans struct {
		Start, Stop, Stable, Update chan *DeployablePair
		Errs                        chan *DiffResolution
		sync.WaitGroup
	}

	// A DeployablePair is a pair of deployables, describing a "before and after"
	// situation, where the Prior Deployable is the known state and the Post
	// Deployable is the desired state.
	DeployablePair struct {
		Diffs        Differences
		Prior, Post  *Deployable
		name         DeploymentID
		ExecutorData interface{}
	}

	// DeployablePairKind describes the disposition of a DeployablePair
	DeployablePairKind int

	diffSet struct {
		New, Gone, Same, Changed DeployablePairs
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
		Start:  make(chan *DeployablePair, s),
		Stop:   make(chan *DeployablePair, s),
		Stable: make(chan *DeployablePair, s),
		Update: make(chan *DeployablePair, s),
		Errs:   make(chan *DiffResolution, s),
	}
}

// Close closes all the channels in a DeployableChans in a single action
func (d *DeployableChans) Close() {
	close(d.Start)
	close(d.Stop)
	close(d.Stable)
	close(d.Update)
	close(d.Errs)
}

func (d *DeployableChans) collect() diffSet {
	ds := newDiffSet()

	for n := range d.Start {
		ds.New = append(ds.New, n)
	}
	for g := range d.Stop {
		ds.Gone = append(ds.Gone, g)
	}
	for s := range d.Stable {
		ds.Same = append(ds.Same, s)
	}
	for m := range d.Update {
		ds.Changed = append(ds.Changed, m)
	}
	return ds
}

// ID returns the ID of this DeployablePair.
func (dp *DeployablePair) ID() DeploymentID {
	return dp.name
}

// Kind returns the kind of the pair
func (dp *DeployablePair) Kind() DeployablePairKind {
	switch {
	case dp.Prior == nil:
		return AddedKind
	case dp.Post == nil:
		return RemovedKind
	case len(dp.Diffs) == 0:
		return SameKind
	default:
		return ModifiedKind
	}
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
