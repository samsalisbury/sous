package sous

import (
	"context"
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
		Prior, Post  *Deployable
		name         DeploymentID
		ExecutorData interface{}
	}

	diffSet struct {
		New, Gone, Same, Changed DeployablePairs
	}
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

// DeployableProcessor processes DeployablePairs off of a DeployableChans channel
type DeployableProcessor interface {
	Start(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Stop(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Stable(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Update(dp *DeployablePair) (*DeployablePair, *DiffResolution)
}

// Pipeline attaches a DeployableProcessor to the DeployableChans, and returns a new DeployableChans.
func (d *DeployableChans) Pipeline(ctx context.Context, proc DeployableProcessor) *DeployableChans {
	out := NewDeployableChans(1)

	wg := sync.WaitGroup{}
	wg.Add(4)

	go func() {
		for rez := range d.Errs {
			out.Errs <- rez
		}
		wg.Wait()
		close(out.Errs)
	}()

	process := func(from, to chan *DeployablePair, doProc func(dp *DeployablePair) (*DeployablePair, *DiffResolution)) {
		defer close(to)
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case dp, ok := <-from:
				if ok {
					proked, rez := doProc(dp)
					if rez != nil {
						out.Errs <- rez
					}
					if proked != nil {
						to <- proked
					}
				} else {
					return
				}
			}
		}
	}

	go process(d.Start, out.Start, proc.Start)
	go process(d.Stop, out.Stop, proc.Stop)
	go process(d.Stable, out.Stable, proc.Stable)
	go process(d.Update, out.Update, proc.Update)
	return out
}

// DeployablePassThrough implements DeployableProcessor trivially: each method simply passes its argument on
type DeployablePassThrough struct{}

// Start returns its argument
func (DeployablePassThrough) Start(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	return dp, nil
}

// Stop returns its argument
func (DeployablePassThrough) Stop(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	return dp, nil
}

// Stable returns its argument
func (DeployablePassThrough) Stable(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	return dp, nil
}

// Update returns its argument
func (DeployablePassThrough) Update(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	return dp, nil
}
