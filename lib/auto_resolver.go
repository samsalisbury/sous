package sous

import (
	"time"
)

type (
	triggerType     struct{}
	triggerChannel  chan triggerType
	announceChannel chan error

	AutoResolveListener func(tc, done triggerChannel, ac announceChannel)

	// An AutoResolver sets up the interactions to automatically run an infinite loop
	// of resolution cycles
	AutoResolver struct {
		UpdateTime time.Duration
		StateReader
		*Resolver
		*LogSet
		listeners []AutoResolveListener
	}
)

func (tc triggerChannel) trigger() {
	tc <- triggerType{}
}

// NewAutoResolver creates a new AutoResolver
func NewAutoResolver(rez *Resolver, sr StateReader, ls *LogSet) *AutoResolver {
	ar := &AutoResolver{
		UpdateTime:  60 * time.Second,
		Resolver:    rez,
		StateReader: sr,
		LogSet:      ls,
		listeners:   make([]AutoResolveListener, 0),
	}
	ar.StandardListeners()
	return ar
}

// StandardListeners adds the usual listeners into the auto-resolve cycle
func (ar *AutoResolver) StandardListeners() {
	ar.addListener(func(trigger, done triggerChannel, ch announceChannel) {
		ar.afterDone(trigger, done, ch)
	})
	ar.addListener(func(trigger, done triggerChannel, ch announceChannel) {
		ar.errorLogging(trigger, done, ch)
	})
}

func (ar *AutoResolver) addListener(f AutoResolveListener) {
	ar.listeners = append(ar.listeners, f)
}

// Kickoff starts the auto-resolve cycle
func (ar *AutoResolver) Kickoff() triggerChannel {
	trigger := make(triggerChannel)
	announce := make(announceChannel)
	done := make(triggerChannel)

	var fanout []announceChannel

	go loopTilDone(func() {
		ar.resolveLoop(trigger, done, announce)
	}, done)

	for _, tf := range ar.listeners {
		ch := make(announceChannel)
		fanout = append(fanout, ch)
		go func(f AutoResolveListener, ch announceChannel) {
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
	for {
		select {
		default:
			ar.LogSet.Debug.Print("Beginning Resolve")
			state, err := ar.StateReader.ReadState()
			ar.LogSet.Debug.Printf("Reading current state: err: %v", err)
			if err != nil {
				ac <- err
				break
			}
			gdm, err := state.Deployments()
			ar.LogSet.Debug.Printf("Reading GDM from state: err: %v", err)

			if err != nil {
				ac <- err
				break
			}

			ac <- ar.Resolver.Resolve(gdm, state.Defs.Clusters)
			ar.LogSet.Debug.Print("Completed resolve")
		case <-done:
			return
		case t := <-tc:
			ar.LogSet.Debug.Printf("Received extra trigger before starting Resolve: %v", t)
			continue
		}

		break
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
