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
func (r *Rectification) Begin(d Deployer) {
	r.once.Do(func() {
		// Right now we are just fudging along with the current channel-based
		// interface of Deployer.Rectify; we can probably make that interface
		// simpler now.
		c := make(chan *DeployablePair, 1)
		c <- &r.Pair
		close(c)
		res := make(chan DiffResolution)
		defer close(res)
		go d.Rectify(c, res)
		// Set Resolution for later querying.
		r.Resolution = <-res
		close(r.done)
	})
}

// Wait must be called after Begin. It waits for and returns the result.
func (r *Rectification) Wait() DiffResolution {
	<-r.done
	return r.Resolution
}
