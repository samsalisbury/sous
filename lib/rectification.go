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
		r.Resolution = d.Rectify(&r.Pair)
		// TODO SS: This select statement is a bandage around the problem
		// that somehow this channel is being closed before reaching the line
		// below. I doubt it's a bug in sync.Once (though that should be
		// eliminated). We should figure out the cause and remove this select.
	CLOSE_CH:
		select {
		default:
			close(r.done)
		case _, open := <-r.done:
			if open {
				goto CLOSE_CH
			}
			// Already closed.
		}
	})
}

// Wait must be called after Begin. It waits for and returns the result.
func (r *Rectification) Wait() DiffResolution {
	<-r.done
	return r.Resolution
}
