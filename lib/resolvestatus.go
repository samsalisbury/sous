package sous

import "sync"

type (
	// ResolveStatus captures the status of a Resolve
	ResolveStatus struct {
		Phase string
		Log   []DiffResolution
		Errs  ResolveErrors
	}

	// ResolveRecorder represents the status of a resolve run.
	ResolveRecorder struct {
		status *ResolveStatus
		// Log is a channel of statuses of individual diff resolutions.
		Log chan DiffResolution
		// finished may be closed with no error, or closed after a single
		// error is emitted to the channel.
		finished chan struct{}
		// err is the final error returned from a phase that ends the resolution.
		err error
		sync.RWMutex
	}

	// DiffResolution is the result of applying a single diff.
	DiffResolution struct {
		DeployID DeployID
		Desc     string
		Error    error
	}
)

// NewResolveRecorder creates a new ResolveRecorder and calls f with it as its
// argument. It then returns that ResolveRecorder immediately.
func NewResolveRecorder(f func(*ResolveRecorder)) *ResolveRecorder {
	rr := &ResolveRecorder{
		status: &ResolveStatus{
			Log:  []DiffResolution{},
			Errs: ResolveErrors{Causes: []error{}},
		},
		Log:      make(chan DiffResolution, 1e6),
		finished: make(chan struct{}),
	}

	go func() {
		for rez := range rr.Log {
			rr.write(func() {
				rr.status.Log = append(rr.status.Log, rez)
				if rez.Error != nil {
					rr.status.Errs.Causes = append(rr.status.Errs.Causes, rez.Error)
					Log.Debug.Printf("resolve error = %+v\n", rez.Error)
				}
			})
		}
	}()

	go func() {
		f(rr)
		close(rr.Log)
		rr.write(func() {
			select {
			default:
				close(rr.finished)
			case _, open := <-rr.finished:
				if open {
					close(rr.finished)
				}
			}
			if rr.err == nil {
				rr.status.Phase = "finished"
			}
		})
	}()
	return rr
}

// Err returns any collected error from the course of resolution
func (rs *ResolveStatus) Err() error {
	if len(rs.Errs.Causes) > 0 {
		return &rs.Errs
	}
	return nil
}

func (rr *ResolveRecorder) foldErrors(log chan DiffResolution) error {
	re := &ResolveErrors{Causes: []error{}}
	for err := range log {
		if err.Error != nil {
			re.Causes = append(re.Causes, err.Error)
			Log.Debug.Printf("resolve error = %+v\n", err)
		}
	}

	if len(re.Causes) > 0 {
		return re
	}
	return nil
}

// Done returns true if the resolution has finished. Otherwise it returns false.
func (rr *ResolveRecorder) Done() bool {
	select {
	case <-rr.finished:
		return true
	default:
		return false
	}
}

// Wait blocks until the resolution is finished.
func (rr *ResolveRecorder) Wait() error {
	<-rr.finished
	var err error
	rr.read(func() { err = rr.err })
	if err != nil {
		return err
	}
	return rr.status.Err()
}

// performPhase performs the requested phase, only if nothing has cancelled the
// resolve.
func (rr *ResolveRecorder) performPhase(name string, f func() error) {
	if rr.Done() {
		return
	}
	rr.setPhase(name)
	if err := f(); err != nil {
		rr.doneWithError(err)
	}
}

func (rr *ResolveRecorder) performGuaranteedPhase(name string, f func()) {
	rr.performPhase(name, func() error { f(); return nil })
}

// setPhase sets the phase of this resolve status.
func (rr *ResolveRecorder) setPhase(phase string) {
	rr.write(func() {
		rr.status.Phase = phase
	})
}

// Phase returns the name of the current phase.
func (rr *ResolveRecorder) Phase() string {
	var phase string
	rr.read(func() { phase = rr.status.Phase })
	return phase
}

// write encapsulates locking this ResolveRecorder for writing using f.
func (rr *ResolveRecorder) write(f func()) {
	rr.Lock()
	defer rr.Unlock()
	f()
}

// read encapsulates locking this ResolveRecorder for reading using f.
func (rr *ResolveRecorder) read(f func()) {
	rr.RLock()
	defer rr.RUnlock()
	f()
}

// doneWithError marks the resolution as finished with an error.
func (rr *ResolveRecorder) doneWithError(err error) {
	rr.write(func() {
		rr.err = err
		close(rr.finished)
	})
}

// done marks the resolution as done.
func (rr *ResolveRecorder) done() {
	rr.write(func() {
		close(rr.finished)
	})
}
