package sous

import (
	"context"
	"sync"
)

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
			case dp, ok := <-from:
				if ok {
					proked, rez := doProc(dp)
					if rez != nil {
						handle(rez)
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

	go func() {
		for rez := range d.Errs {
			handle(rez)
			out.Errs <- rez
		}
		wg.Wait()
		close(out.Errs)
		d.Done()
	}()

	go process(d.Start, out.Start, proc.Start)
	go process(d.Stop, out.Stop, proc.Stop)
	go process(d.Stable, out.Stable, proc.Stable)
	go process(d.Update, out.Update, proc.Update)
	return out
}

func nullHandler(err *DiffResolution) {}

// DeployableProcessor processes DeployablePairs off of a DeployableChans channel
type DeployableProcessor interface {
	Start(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Stop(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Stable(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Update(dp *DeployablePair) (*DeployablePair, *DiffResolution)
}

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
