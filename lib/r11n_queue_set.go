package sous

import "sync"

// R11nQueueSet is a concurrency-safe mapping of DeploymentID to R11nQueue.
type R11nQueueSet struct {
	set  map[DeploymentID]*R11nQueue
	opts []R11nQueueOpt
	sync.RWMutex
}

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
