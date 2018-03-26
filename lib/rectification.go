package sous

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Rectification represents the rectification of a single DeployablePair.
type Rectification struct {
	// Pair is not a pointer as it's considered an immutable instruction.
	Pair DeployablePair
	// Resolution is the final resolution of this single rectification.
	sync.RWMutex
	Resolution DiffResolution

	once   sync.Once
	ctx    context.Context
	cancel func()
}

// NewRectification is used to rectify differences on a single Deployment.
// After this its useful life is over.
func NewRectification(dp DeployablePair) *Rectification {
	c, cancel := context.WithCancel(context.Background())
	return &Rectification{
		Pair:   dp,
		ctx:    c,
		cancel: cancel,
	}
}

// Begin begins applying sr.Pair using d Deployer. Call Result to get the
// result. Begin can be called multiple times but performs its function only
// once.
func (r *Rectification) Begin(d Deployer, reg Registry, rf *ResolveFilter, stateReader StateReader) {
	r.once.Do(func() {
		go func() {
			defer r.cancel()
			if r.Pair.Post.BuildArtifact == nil {
				pair, diff := HandlePairsByRegistry(reg, &r.Pair)
				if diff != nil && diff.Error != nil {
					r.Lock()
					r.Resolution.Error = WrapResolveError(diff.Error)
					r.Unlock()
					return
				}
				if pair != nil {
					r.Pair = *pair
				} else {
					r.Lock()
					r.Resolution.Error = WrapResolveError(fmt.Errorf("Unknown Error Occurred, no resolve error and no pair present"))
					r.Unlock()
					return
				}
			}
			r.Lock()
			r.Resolution = d.Rectify(&r.Pair)
			r.Unlock()

			state, err := stateReader.ReadState()
			if err != nil {
				r.Lock()
				r.Resolution.Error = WrapResolveError(err)
				r.Unlock()
				return
			}

			clusters := state.Defs.Clusters
			clusters = rf.FilteredClusters(clusters)

			// TODO constants / configs
			tick := time.NewTicker(250 * time.Millisecond)
			defer tick.Stop()

			end, ec := context.WithTimeout(r.ctx, 20*time.Minute)
			defer ec()

			for {
				s, err := r.pollOnce(d, reg, clusters)
				if err != nil {
					r.Lock()
					r.Resolution.Error = &ErrorWrapper{error: err}
					r.Unlock()
					return
				}
				if s.Final() && s.SourceID.Equal(r.Pair.Post.SourceID) {
					r.Lock()
					r.Resolution.DeployState = s
					r.Unlock()
					return
				}
				select {
				case <-tick.C:
				case <-end.Done():
					r.Lock()
					defer r.Unlock()
					if r.Resolution.DeployState == nil {
						r.Resolution.DeployState = &DeployState{}
					}
					return
				}
			}

		}()
	})
}

func (r *Rectification) pollOnce(d Deployer, reg Registry, clusters Clusters) (*DeployState, error) {
	// XXX thread the context from Begin into Deployer.Status
	depState, err := d.Status(reg, clusters, &r.Pair)
	if err != nil {
		return nil, err
	}
	return depState, nil
}

// Wait must be called after Begin. It waits for and returns the result.
func (r *Rectification) Wait() DiffResolution {
	<-r.ctx.Done()
	r.RLock()
	defer r.RUnlock()
	return r.Resolution
}
