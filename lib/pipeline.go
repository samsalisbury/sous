package sous

import (
	"context"
)

// DeployableProcessor processes DeployablePairs off of a DeployableChans channel
type DeployableProcessor interface {
	HandlePairs(dp *DeployablePair) (*DeployablePair, *DiffResolution)
}

// Pipeline attaches a DeployableProcessor to the DeployableChans, and returns a new DeployableChans.
// Each segment of a pipeline consumes from
//   a channel of DeployablePairs and
//   a channel of error
// and likewise produceds into a similar pair of channels.
// Each 'Pair that comes over the channel is handled to the DeployableProcessor to be processed.
// Processed pairs are put into the output channel,
//   and errors from the processing go into the output error channel.
// Incoming errors are repeated onto the output error channel as well.
// The DeployableProcessor can optionally provide a HandleResolution method,
// in which case processing errors (both its own and from upstream) will be provided
// to that method. HandleResolution, by interface, has no return:
// the errors proceed unconditionally to the output error channel regardless.
func (d *DeployableChans) Pipeline(ctx context.Context, proc DeployableProcessor) *DeployableChans {
	out := NewDeployableChans(1)

	handle := nullHandler
	if handler, is := proc.(DeployableResolutionHandler); is {
		handle = handler.HandleResolution
	}

	d.Add(2) // for the errs channel

	transErrs := make(chan *DiffResolution)

	go func(from, to chan *DeployablePair, doProc func(dp *DeployablePair) (*DeployablePair, *DiffResolution)) {
		defer close(to)
		defer close(transErrs)
		defer d.Done()
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
					transErrs <- rez
				}
				if proked != nil {
					to <- proked
				}
			}
		}
	}(d.Pairs, out.Pairs, proc.HandlePairs)

	go func(upstream, local <-chan *DiffResolution) {
		defer close(out.Errs)
		defer d.Done()

		for {
			if upstream == nil && local == nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			case rez, open := <-upstream:
				if !open {
					upstream = nil
				}
				if rez != nil {
					handle(rez)
					out.Errs <- rez
				}
			case rez, open := <-local:
				if !open {
					local = nil
				}
				if rez != nil {
					handle(rez)
					out.Errs <- rez
				}
			}
		}
	}(d.Errs, transErrs)

	return out
}

func nullHandler(err *DiffResolution) {}

// DeployableResolutionHandler handles the resolution of a single DiffResolution.
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
