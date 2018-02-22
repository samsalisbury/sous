package sous

import (
	"context"
	"sync"

	"github.com/opentable/sous/util/logging"
)

type (
	// A DeployableChans is a bundle of channels describing actions to take on a
	// cluster
	DeployableChans struct {
		Pairs chan *DeployablePair
		Errs  chan *DiffResolution
		sync.WaitGroup
	}

	// DeployablePairs is a list of DeployablePair
	DeployablePairs []*DeployablePair
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

func (d *DeployableChans) collect() DeployablePairs {
	ds := DeployablePairs{}

	for m := range d.Pairs {
		ds = append(ds, m)
	}
	return ds
}

// Collect returns a collected list of DeployablePairs represented by this DeployableChans
func (d *DeployableChans) Collect() DeployablePairs {
	return d.collect()
}

// ID returns the ID of this DeployablePair.
func (dp *DeployablePair) ID() DeploymentID {
	return dp.name
}

// SetID sets the ID of this DeployablePair. Do not use except in tests.
func (dp *DeployablePair) SetID(did DeploymentID) {
	dp.name = did
}

// Log adds a logging pipeline step onto a DeployableChans
func (d *DeployableChans) Log(ctx context.Context, ls logging.LogSink) *DeployableChans {
	proc := loggingProcessor{ls: ls}
	return d.Pipeline(ctx, proc)
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

// ExpectedResolutionType returns the expected resolution for this kind.
// This is used for logging purposes, when we drop a diff and don't attempt
// to rectify it.
func (kind DeployablePairKind) ExpectedResolutionType() ResolutionType {
	switch kind {
	default:
		return "unknown"
	case SameKind:
		return StableDiff
	case AddedKind:
		// Note: we never return ComingDiff as that's an intermediate state.
		return CreateDiff
	case RemovedKind:
		return DeleteDiff
	case ModifiedKind:
		return ModifyDiff
	}
}

// ResolveVerb provides an imperative verb describing the resolution action
// required to resolve this kind of deployable pair.
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
