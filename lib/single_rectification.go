package sous

// SingleRectification shepherds through a rectification for a single
// DeployablePair. After that, it should be discarded.
type SingleRectification struct {
	// DeployablePair is not a pointer as considered an immutable instruction.
	Pair DeployablePair
	// Resolution is the final resolution of this single rectification.
	Resolution DiffResolution
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

// Resolve synchronously applies sr.Pair using d Deployer and returns the result.
func (sr *SingleRectification) Resolve(d Deployer) DiffResolution {
	// Right now we are just fudging along with the current channel-based
	// interface of Deployer.Rectify; we can probably make that interface
	// simpler now.
	c := make(chan *DeployablePair, 1)
	defer close(c)
	c <- &sr.Pair
	r := make(chan DiffResolution)
	defer close(r)
	d.Rectify(c, r)
	// Set Resolution for later querying.
	sr.Resolution = <-r
	close(sr.done)
	return sr.Resolution
}

// Wait waits until this rectification is done and returns the resolution.
func (sr *SingleRectification) Wait() DiffResolution {
	<-sr.done
	return sr.Resolution
}
