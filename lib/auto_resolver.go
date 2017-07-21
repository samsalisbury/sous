package sous

import (
	"sync"
	"time"

	"github.com/opentable/sous/util/logging"
)

type (
	// TriggerType is an empty struct, representing some kind of trigger.
	TriggerType struct{}
	// TriggerChannel is a channel of TriggerType.
	TriggerChannel  chan TriggerType
	announceChannel chan error

	// autoResolveListener listens to trigger channels and writes to announceChannel.
	autoResolveListener func(tc, done TriggerChannel, ac announceChannel)

	// An AutoResolver sets up the interactions to automatically run an infinite
	// loop of resolution cycles.
	AutoResolver struct {
		UpdateTime time.Duration
		StateReader
		GDM Deployments
		*Resolver
		*logging.LogSet
		listeners []autoResolveListener
		sync.RWMutex
		stableStatus, liveStatus *ResolveStatus
		currentRecorder          *ResolveRecorder
	}
)

func (tc TriggerChannel) trigger() {
	tc <- TriggerType{}
}

// NewAutoResolver creates a new AutoResolver.
func NewAutoResolver(rez *Resolver, sr StateReader, ls *logging.LogSet) *AutoResolver {
	ar := &AutoResolver{
		UpdateTime:  60 * time.Second,
		Resolver:    rez,
		StateReader: sr,
		LogSet:      ls,
		listeners:   make([]autoResolveListener, 0),
	}
	ar.StandardListeners()
	return ar
}

// StandardListeners adds the usual listeners into the auto-resolve cycle.
func (ar *AutoResolver) StandardListeners() {
	ar.addListener(func(trigger, done TriggerChannel, ch announceChannel) {
		ar.afterDone(trigger, done, ch)
	})
	ar.addListener(func(trigger, done TriggerChannel, ch announceChannel) {
		ar.errorLogging(trigger, done, ch)
	})
}

func (ar *AutoResolver) addListener(f autoResolveListener) {
	ar.listeners = append(ar.listeners, f)
}

// Kickoff starts the auto-resolve cycle.
func (ar *AutoResolver) Kickoff() TriggerChannel {
	trigger := make(TriggerChannel)
	announce := make(announceChannel)
	done := make(TriggerChannel)

	var fanout []announceChannel

	go loopTilDone(func() {
		ar.resolveLoop(trigger, done, announce)
	}, done)

	for _, tf := range ar.listeners {
		ch := make(announceChannel)
		fanout = append(fanout, ch)
		go func(f autoResolveListener, ch announceChannel) {
			loopTilDone(func() {
				f(trigger, done, ch)
			}, done)
		}(tf, ch)
	}

	go loopTilDone(func() {
		ar.multicast(done, announce, fanout)
	}, done)
	trigger.trigger()

	return done
}

func (ar *AutoResolver) updateStatus() {
	if ar.currentRecorder == nil {
		return
	}
	ar.write(func() {
		ls := ar.currentRecorder.CurrentStatus()
		logging.Log.Debugf("Recording live status from %p: %v", ar, ls)
		ar.liveStatus = &ls
	})
}

// Statuses returns the current status of the resolution underway.
// The returned statuses are "stable" - the unchanging, complete status of the previous resolve
// and "live" - the current status of the running resolution
func (ar *AutoResolver) Statuses() (stable, live *ResolveStatus) {
	ar.updateStatus()
	ar.RLock()
	defer ar.RUnlock()
	logging.Log.Debugf("Reporting statuses from %p: %v %v", ar, ar.stableStatus, ar.liveStatus)
	return ar.stableStatus, ar.liveStatus
}

func loopTilDone(f func(), done TriggerChannel) {
	for {
		select {
		default:
			f()
		case <-done:
			return
		}
	}
}

func (ar *AutoResolver) write(f func()) {
	logging.Log.Vomitf("Locking autoresolver for write...")
	ar.Lock()
	defer func() {
		ar.Unlock()
		logging.Log.Vomitf("Unlocked autoresolver")
	}()
	f()
}

func (ar *AutoResolver) resolveLoop(tc, done TriggerChannel, ac announceChannel) {
	select {
	case <-done:
		return
	case <-tc:
	}
	for {
		select {
		default:
			ar.resolveOnce(ac)
		case <-done:
			return
		case t := <-tc:
			ar.LogSet.Debugf("Received extra trigger before starting Resolve: %v", t)
			continue
		}

		break
	}
}

func (ar *AutoResolver) resolveOnce(ac announceChannel) {
	ar.LogSet.Debugf("Beginning Resolve")
	state, err := ar.StateReader.ReadState()
	ar.LogSet.Debugf("Reading current state: err: %v", err)
	if err != nil {
		ac <- err
		return
	}
	ar.GDM, err = state.Deployments()
	ar.LogSet.Debugf("Reading GDM from state: err: %v", err)

	if err != nil {
		ac <- err
		return
	}

	ar.write(func() {
		ar.currentRecorder = ar.Resolver.Begin(ar.GDM, state.Defs.Clusters)
	})
	defer ar.write(func() {
		ar.currentRecorder = nil
	})
	ac <- ar.currentRecorder.Wait()
	ar.write(func() {
		ss := ar.currentRecorder.CurrentStatus()
		logging.Log.Debugf("Recording stable status from %p: %v", ar, ss)

		if ss.Started.Before(ss.Finished) {
			ar.LogSet.GetTimer("fullcycle-duration").Update(ss.Finished.Sub(ss.Started))
		} else {
			ar.LogSet.Warnf("No finished time recorded for supposed stable status.")
		}

		ar.stableStatus = &ss
	})
	ar.Statuses() // XXX this is debugging
	ar.LogSet.Debugf("Completed resolve")
}

func (ar *AutoResolver) afterDone(tc, done TriggerChannel, ac announceChannel) {
	select {
	case <-done:
		return
	case <-ac:
	}
	select {
	case <-done:
		return
	case <-time.After(ar.UpdateTime):
	}
	tc.trigger()
}

func (ar *AutoResolver) errorLogging(tc, done TriggerChannel, errs announceChannel) {
	select {
	case <-done:
		return
	case e := <-errs:
		if e != nil {
			ar.LogSet.Warnf("error:", e)
		}
	}
}

func (ar *AutoResolver) multicast(done TriggerChannel, ac announceChannel, fo []announceChannel) {
	select {
	case <-done:
		return
	case res := <-ac:
		for _, c := range fo {
			c <- res
		}
	}
}
