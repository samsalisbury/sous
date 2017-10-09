package sous

import (
	"context"
	"sync"
)

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

// DeployableProcessor processes DeployablePairs off of a DeployableChans channel
type DeployableProcessor interface {
	Start(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Stop(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Stable(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Update(dp *DeployablePair) (*DeployablePair, *DiffResolution)
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
