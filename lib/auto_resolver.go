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
		Clusters
		StateReader
		*Resolver
		*LogSet
	}
)

func (tc triggerChannel) trigger() {
	tc <- triggerType{}
}

func (ar *AutoResolver) kickoff() {
	trigger := make(triggerChannel)
	announce := make(announceChannel)
	var fanout []announceChannel

	go ar.resolveLoop(trigger, announce)

	triggerFuncs := []func(announceChannel){
		func(ch announceChannel) { ar.afterDone(60*time.Second, trigger, ch) },
		func(ch announceChannel) { ar.errorLogging(ch) },
	}
	for _, f := range triggerFuncs {
		ch := make(announceChannel)
		fanout = append(fanout, ch)
		go f(ch)
	}

	go ar.multicast(announce, fanout)
	trigger.trigger()
}

func (ar *AutoResolver) resolveLoop(tc triggerChannel, ac announceChannel) {
	for {
		_ = <-tc
		select {
		default:
			state, err := ar.StateReader.ReadState()
			if err != nil {
				ac <- err
				continue
			}
			gdm, err := state.Deployments()
			if err != nil {
				ac <- err
				continue
			}

			ac <- ar.Resolver.Resolve(gdm, state.Defs.Clusters)
		case _ = <-tc:
		}
	}
}

func (ar *AutoResolver) afterDone(w time.Duration, tc triggerChannel, ac announceChannel) {
	for {
		_ = <-ac
		_ = <-time.After(w)
		tc.trigger()
	}
}

func (ar *AutoResolver) errorLogging(errs announceChannel) {
	for {
		if e := <-errs; e != nil {
			ar.LogSet.Warn.Print(e)
		}
	}
}

func (ar *AutoResolver) multicast(ac announceChannel, fo []announceChannel) {
	for {
		err := <-ac
		for _, c := range fo {
			c <- err
		}
	}
}
