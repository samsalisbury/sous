package sous

import "sync"

// Rectification represents the rectification of a single DeployablePair.
type Rectification struct {
	// Pair is not a pointer as it's considered an immutable instruction.
	Pair DeployablePair
	// Resolution is the final resolution of this single rectification.
	Resolution DiffResolution
	once       sync.Once
	done       chan struct{}
}

// NewRectification is used to rectify differences on a single Deployment.
// After this its useful life is over.
func NewRectification(dp DeployablePair) *Rectification {
	return &Rectification{
		Pair: dp,
		done: make(chan struct{}),
	}
}

// Begin begins applying sr.Pair using d Deployer. Call Result to get the
// result. Begin can be called multiple times but performs its function only
// once.
func (r *Rectification) Begin(d Deployer, reg Registry, rf *ResolveFilter, stateReader StateReader) {
	r.once.Do(func() {
		// For it to be sensible to separate Begin and Wait,
		// this needs to happen async
		go func() {
			defer close(r.done)
			if r.Pair.Post.BuildArtifact == nil {
				pair, _ := HandlePairsByRegistry(reg, &r.Pair)
				r.Pair = *pair
			}
			r.Resolution = d.Rectify(&r.Pair)

			state, err := stateReader.ReadState()
			if err != nil {
				r.Resolution.Error = WrapResolveError(err)
				return
			}

			clusters := state.Defs.Clusters
			clusters = rf.FilteredClusters(clusters)

			depState, err := d.Status(reg, clusters, &r.Pair)
			if err != nil {
				r.Resolution.Error = WrapResolveError(err)
				return
			}
			r.Resolution.DeployState = depState

		}()
	})
}

// Wait must be called after Begin. It waits for and returns the result.
func (r *Rectification) Wait() DiffResolution {
	<-r.done
	return r.Resolution
}
