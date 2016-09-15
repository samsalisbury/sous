package sous

import (
	"time"
)

type (
	triggerType     struct{}
	triggerChannel  chan triggerType
	announceChannel chan error

	// An AutoResolver sets up the interactions to automatically run an infinite loop
	// of resolution cycles
	AutoResolver struct {
		UpdateTime time.Duration
		Clusters
		StateReader
		*Resolver
		*LogSet
	}
)

func (tc triggerChannel) trigger() {
	tc <- triggerType{}
}

func (ar *AutoResolver) kickoff() triggerChannel {
	trigger := make(triggerChannel)
	announce := make(announceChannel)
	done := make(triggerChannel)

	var fanout []announceChannel

	go loopTilDone(func() {
		ar.resolveLoop(trigger, done, announce)
	}, done)

	triggerFuncs := []func(announceChannel){
		func(ch announceChannel) { ar.afterDone(trigger, done, ch) },
		func(ch announceChannel) { ar.errorLogging(trigger, done, ch) },
	}
	for _, f := range triggerFuncs {
		ch := make(announceChannel)
		fanout = append(fanout, ch)
		go func(f func(announceChannel), ch announceChannel) {
			loopTilDone(func() {
				f(ch)
			}, done)
		}(f, ch)
	}

	go loopTilDone(func() {
		ar.multicast(done, announce, fanout)
	}, done)
	trigger.trigger()

	return done
}

func loopTilDone(f func(), done triggerChannel) {
	for {
		select {
		default:
			f()
		case <-done:
			return
		}
	}
}

func (ar *AutoResolver) resolveLoop(tc, done triggerChannel, ac announceChannel) {
	select {
	case <-done:
		return
	case <-tc:
	}
	select {
	default:
		state, err := ar.StateReader.ReadState()
		if err != nil {
			ac <- err
			return
		}
		gdm, err := state.Deployments()
		if err != nil {
			ac <- err
			return
		}

		ac <- ar.Resolver.Resolve(gdm, state.Defs.Clusters)
	case <-done:
		return
	case <-tc:
	}
}

func (ar *AutoResolver) afterDone(tc, done triggerChannel, ac announceChannel) {
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

func (ar *AutoResolver) errorLogging(tc, done triggerChannel, errs announceChannel) {
	select {
	case <-done:
		return
	case e := <-errs:
		if e != nil {
			ar.LogSet.Warn.Print(e)
		}
	}
}

func (ar *AutoResolver) multicast(done triggerChannel, ac announceChannel, fo []announceChannel) {
	select {
	case <-done:
		return
	case res := <-ac:
		for _, c := range fo {
			c <- res
		}
	}
}
