package sous

// SingleRectification shepherds through a rectification for a single
// DeployablePair. After that, it should be discarded.
type SingleRectification struct {
	// DeployablePair is not a pointer as considered an immutable instruction.
	Pair DeployablePair
	// Resolution is the final resolution of this single rectification.
	Resolution DiffResolution
}

// NewSingleRectification is used to rectify differences on a single Deployment.
// After this its useful life is over.
func NewSingleRectification(dp DeployablePair) *SingleRectification {
	return &SingleRectification{
		Pair: dp,
	}
}

// Resolve synchronously applies sr.Pair using d Deployer and returns the result.
func (sr *SingleRectification) Resolve(d Deployer) DiffResolution {
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
	return sr.Resolution
}
