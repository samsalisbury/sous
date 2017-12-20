package sous

import "sync"

// SingleRectification shepherds through a rectification for a single
// DeployablePair. After that, it should be discarded.
type SingleRectification struct {
	// DeployablePair is not a pointer as considered an immutable instruction.
	Pair DeployablePair
	// Resolution is the final resolution of this single rectification.
	Resolution DiffResolution
	once       sync.Once
	done       chan struct{}
}

// NewSingleRectification is used to rectify differences on a single Deployment.
// After this its useful life is over.
func NewSingleRectification(dp DeployablePair) *SingleRectification {
	return &SingleRectification{
		Pair: dp,
		done: make(chan struct{}),
	}
}

// Begin begins applying sr.Pair using d Deployer. Call Result to get the
// result. Begin can be called multiple times but performs its function only
// once.
func (sr *SingleRectification) Begin(d Deployer) {
	sr.once.Do(func() {
		// Right now we are just fudging along with the current channel-based
		// interface of Deployer.Rectify; we can probably make that interface
		// simpler now.
		c := make(chan *DeployablePair, 1)
		c <- &sr.Pair
		close(c)
		r := make(chan DiffResolution)
		defer close(r)
		go d.Rectify(c, r)
		// Set Resolution for later querying.
		sr.Resolution = <-r
		close(sr.done)
	})
}

// Wait must be called after Begin. It waits for and returns the result.
func (sr *SingleRectification) Wait() DiffResolution {
	<-sr.done
	return sr.Resolution
}
