package sous

import (
	"sync"

	"github.com/nyarly/spies"
)

type (
	// QueueSet is the interface for a set of queues
	QueueSet interface {
		PushIfEmpty(r *Rectification) (*QueuedR11n, bool)
		Push(r *Rectification) (*QueuedR11n, bool)
		Wait(did DeploymentID, id R11nID) (DiffResolution, bool)
		Queues() map[DeploymentID]*R11nQueue
	}

	// R11nQueueSet is a concurrency-safe mapping of DeploymentID to R11nQueue.
	R11nQueueSet struct {
		set  map[DeploymentID]*R11nQueue
		opts []R11nQueueOpt
		reg  Registry
		sync.RWMutex
	}

	// QueueSetSpy is a spy for the QueueSet interface.
	QueueSetSpy struct {
		*spies.Spy
	}
)

// NewR11nQueueSet returns a ready to use R11nQueueSet.
func NewR11nQueueSet(opts ...R11nQueueOpt) *R11nQueueSet {
	return &R11nQueueSet{
		set:  map[DeploymentID]*R11nQueue{},
		opts: opts,
	}
}

// PushIfEmpty creates a queue for the DeploymentID of r if it does not already
// exist. It calls PushIfEmpty on that R11nQueue passing r.
func (rqs *R11nQueueSet) PushIfEmpty(r *Rectification) (*QueuedR11n, bool) {
	rqs.Lock()
	defer rqs.Unlock()
	id := r.Pair.ID()
	queue, ok := rqs.set[id]
	if !ok {
		queue = NewR11nQueue(rqs.opts...)
		rqs.set[id] = queue
	}
	return queue.PushIfEmpty(r)
}

// Push creates a queue for the DeploymentID of r if it does not already
// exist. It calls Push on that R11nQueue passing r.
func (rqs *R11nQueueSet) Push(r *Rectification) (*QueuedR11n, bool) {
	rqs.Lock()
	defer rqs.Unlock()
	id := r.Pair.ID()
	queue, ok := rqs.set[id]
	if !ok {
		queue = NewR11nQueue(rqs.opts...)
		rqs.set[id] = queue
	}
	return queue.Push(r)
}

// Wait waits for the r11n with id id to complete, if it is found in the
// queue for did. If there is no queue for did or it exists but does not contain
// id, then it returns zero DiffResolution, false.
func (rqs *R11nQueueSet) Wait(did DeploymentID, id R11nID) (DiffResolution, bool) {
	rqs.Lock()
	rq, ok := rqs.set[did]
	rqs.Unlock()
	if !ok {
		return DiffResolution{}, false
	}
	return rq.Wait(id)
}

// Queues returns a snapshot of queues in this set.
func (rqs *R11nQueueSet) Queues() map[DeploymentID]*R11nQueue {
	rqs.Lock()
	defer rqs.Unlock()
	s := make(map[DeploymentID]*R11nQueue, len(rqs.set))
	for k, v := range rqs.set {
		s[k] = v
	}
	return s
}

// NewQueueSetSpy builds a spy/controller pair
func NewQueueSetSpy() (QueueSet, *spies.Spy) {
	spy := spies.NewSpy()
	return QueueSetSpy{Spy: spy}, spy
}

// PushIfEmpty is a spy implementation of QueueSet
func (s QueueSetSpy) PushIfEmpty(r *Rectification) (*QueuedR11n, bool) {
	res := s.Called(r)
	return res.Get(0).(*QueuedR11n), res.Bool(1)
}

// Push is a spy implementation of QueueSet
func (s QueueSetSpy) Push(r *Rectification) (*QueuedR11n, bool) {
	res := s.Called(r)
	if res.Get(0) == nil {
		return nil, false
	}
	return res.Get(0).(*QueuedR11n), res.Bool(1)
}

// Wait is a spy implementation of QueueSet
func (s QueueSetSpy) Wait(did DeploymentID, id R11nID) (DiffResolution, bool) {
	res := s.Called(did, id)
	return res.Get(0).(DiffResolution), res.Bool(1)
}

// Queues is a spy implementation of QueueSet
func (s QueueSetSpy) Queues() map[DeploymentID]*R11nQueue {
	res := s.Called()
	return res.Get(0).(map[DeploymentID]*R11nQueue)
}
