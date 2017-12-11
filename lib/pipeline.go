package sous

import (
	"context"
	"sync"
)

// DeployableProcessor processes DeployablePairs off of a DeployableChans channel
type DeployableProcessor interface {
	Pairs(dp *DeployablePair) (*DeployablePair, *DiffResolution)
}

// Pipeline attaches a DeployableProcessor to the DeployableChans, and returns a new DeployableChans.
func (d *DeployableChans) Pipeline(ctx context.Context, proc DeployableProcessor) *DeployableChans {
	out := NewDeployableChans(1)

	handle := nullHandler
	if handler, is := proc.(DeployableResolutionHandler); is {
		handle = handler.HandleResolution
	}

	d.Add(1) // for the errs channel

	wg := sync.WaitGroup{}
	wg.Add(4)

	process := func(from, to chan *DeployablePair, doProc func(dp *DeployablePair) (*DeployablePair, *DiffResolution)) {
		defer close(to)
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case dp, open := <-from:
				if !open {
					return
				}

				proked, rez := doProc(dp)
				if rez != nil {
					handle(rez)
					out.Errs <- rez
				}
				if proked != nil {
					to <- proked
				}
			}
		}
	}

	go func() {
		for rez := range d.Errs {
			handle(rez)
			out.Errs <- rez
		}
		wg.Wait()
		close(out.Errs)
		d.Done()
	}()

	go process(d.Pairs, out.Pairs, proc.Pairs)
	return out
}

func nullHandler(err *DiffResolution) {}

type DeployableResolutionHandler interface {
	HandleResolution(err *DiffResolution)
}

// DeployablePassThrough implements DeployableProcessor trivially: each method simply passes its argument on
// It is intended as an anonymous embed for Processors that e.g. don't care about Stable
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
