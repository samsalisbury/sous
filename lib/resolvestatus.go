package sous

import (
	"fmt"
	"sync"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

type (
	// ResolveStatus captures the status of a Resolve
	ResolveStatus struct {
		// Started collects the time that the Status began being collected
		Started,
		// Finished collects the time that the Status was completed - or the zero
		// time if the status is still live.
		Finished time.Time
		// Phase reports the current phase of resolution
		Phase string
		// Intended are the deployments that are the target of this resolution
		Intended []*Deployment
		// logging.Log collects the resolution steps that have been performed
		Log []DiffResolution
		// Errs collects errors during resolution
		Errs ResolveErrors
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
		logSink logging.LogSink
	}

	// DiffResolution is the result of applying a single diff.
	DiffResolution struct {
		// DeployID is the ID of the deployment being resolved
		DeploymentID
		// Desc describes the difference and its resolution
		Desc ResolutionType
		// Error captures the error (if any) encountered during diff resolution
		Error *ErrorWrapper

		// DeployState is the state of this deployment as running.
		DeployState *DeployState

		// SchedulerURL is a URL where this deployment can be seen.
		SchedulerURL string
	}

	// ResolutionType marks the kind of a DiffResolution
	// XXX should be made an int and generate with gostringer
	ResolutionType string
)

const (
	// StableDiff - the active deployment is the intended deployment
	StableDiff = ResolutionType("unchanged")
	// ComingDiff - the intended deployment is pending, assumed will be come active
	ComingDiff = ResolutionType("coming")
	// CreateDiff - the intended deployment was missing and had to be created.
	CreateDiff = ResolutionType("created")
	// ModifyDiff - there was a deployment that differed from the intended was changed.
	ModifyDiff = ResolutionType("updated")
	// DeleteDiff - a deployment was active that wasn't intended at all, and was deleted.
	DeleteDiff = ResolutionType("deleted")
)

func (rez DiffResolution) String() string {
	return fmt.Sprintf("%s %s %v", rez.DeploymentID, rez.Desc, rez.Error)
}

// EachField implements EachFielder on DiffResolution.
func (rez DiffResolution) EachField(f logging.FieldReportFn) {
	f(logging.SousDeploymentId, rez.DeploymentID.String())
	f(logging.SousManifestId, rez.ManifestID.String())
	f(logging.SousResolutionDescription, string(rez.Desc))
	if rez.Error == nil {
		return
	}
	marshallable := buildMarshableError(rez.Error.error)
	f(logging.SousResolutionErrortype, marshallable.Type)
	f(logging.SousResolutionErrormessage, marshallable.String)
}

// NewResolveRecorder creates a new ResolveRecorder and calls f with it as its
// argument. It then returns that ResolveRecorder immediately.
func NewResolveRecorder(intended Deployments, ls logging.LogSink, f func(*ResolveRecorder)) *ResolveRecorder {
	rr := &ResolveRecorder{
		status: &ResolveStatus{
			Started:  time.Now(),
			Intended: []*Deployment{},
			Log:      []DiffResolution{},
			Errs:     ResolveErrors{Causes: []ErrorWrapper{}},
		},
		Log:      make(chan DiffResolution, 10),
		finished: make(chan struct{}),
		logSink:  ls,
	}

	for _, d := range intended.Snapshot() {
		rr.status.Intended = append(rr.status.Intended, d)
	}

	// Update status incrementally.
	go func() {
		for rez := range rr.Log {
			rr.write(func() {
				rr.status.Log = append(rr.status.Log, rez)
				if rez.Error != nil {
					rr.status.Errs.Causes = append(rr.status.Errs.Causes, ErrorWrapper{error: rez.Error})
					messages.ReportLogFieldsMessage("resolve error", logging.DebugLevel, ls, rez.Error)
				}
			})
		}
	}()

	// Execute the main function (f) over this resolve recorder.
	go func() {
		f(rr)
		close(rr.Log)
		rr.write(func() {
			rr.status.Finished = time.Now()
			if rr.err == nil {
				rr.status.Phase = "finished"
			}
			close(rr.finished)
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

// CurrentStatus returns a copy of the current status of the resolve
func (rr *ResolveRecorder) CurrentStatus() (rs ResolveStatus) {
	rr.read(func() {
		rs = *rr.status
		rs.Log = make([]DiffResolution, len(rr.status.Log))
		copy(rs.Log, rr.status.Log)
		rs.Errs.Causes = make([]ErrorWrapper, len(rr.status.Errs.Causes))
		copy(rs.Errs.Causes, rr.status.Errs.Causes)
	})
	return
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
	rr.read(func() {
		err = rr.err
		if err == nil {
			err = rr.status.Err()
		}
	})
	return err
}

func (rr *ResolveRecorder) earlyExit() (yes bool) {
	rr.read(func() {
		yes = (rr.err != nil)
	})
	return
}

// performPhase performs the requested phase, only if nothing has cancelled the
// resolve.
func (rr *ResolveRecorder) performPhase(name string, f func() error) {
	if rr.earlyExit() {
		messages.ReportLogFieldsMessage("Skipping phase", logging.DebugLevel, rr.logSink, name, rr.err)
		return
	}
	messages.ReportLogFieldsMessage("Performing phase", logging.DebugLevel, rr.logSink, name)
	rr.setPhase(name)
	if err := f(); err != nil {
		rr.doneWithError(err)
	}
}

// setPhase sets the phase of this resolve status.
func (rr *ResolveRecorder) setPhase(phase string) {
	messages.ReportLogFieldsMessage("Setting phase", logging.ExtraDebug1Level, rr.logSink, phase)
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
	name, uuid := logging.RetrieveMetaData(f)
	messages.ReportLogFieldsMessage("Waiting to Lock resolverecorder for write", logging.ExtraDebug1Level, rr.logSink, name, uuid)
	rr.Lock()
	messages.ReportLogFieldsMessage("Locked resolverecorder for write", logging.ExtraDebug1Level, rr.logSink, name, uuid)
	defer func() {
		rr.Unlock()
		messages.ReportLogFieldsMessage("Unlocked resolverecorder for write", logging.ExtraDebug1Level,
			rr.logSink, name, uuid)
	}()
	f()
}

// read encapsulates locking this ResolveRecorder for reading using f.
func (rr *ResolveRecorder) read(f func()) {
	name, uuid := logging.RetrieveMetaData(f)
	messages.ReportLogFieldsMessage("Waiting to Lock resolverecorder for read", logging.ExtraDebug1Level, rr.logSink, name, uuid)
	rr.RLock()
	messages.ReportLogFieldsMessage("Locked resolverecorder for read", logging.ExtraDebug1Level, rr.logSink, name, uuid)
	defer func() {
		rr.RUnlock()
		messages.ReportLogFieldsMessage("Unlocked resolverecorder for read", logging.ExtraDebug1Level,
			rr.logSink, name, uuid)
	}()
	f()
}

// doneWithError marks the resolution as finished with an error.
func (rr *ResolveRecorder) doneWithError(err error) {
	rr.write(func() {
		rr.err = err
	})
}
