package sous

import "sync"

// R11nQueueSet is a concurrency-safe mapping of DeploymentID to R11nQueue.
type R11nQueueSet struct {
	set map[DeploymentID]*R11nQueue
	sync.RWMutex
}

// NewR11nQueueSet returns a ready to use R11nQueueSet.
func NewR11nQueueSet() *R11nQueueSet {
	return &R11nQueueSet{
		set: map[DeploymentID]*R11nQueue{},
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
		queue = NewR11nQueue()
		rqs.set[id] = queue
	}
	return queue.PushIfEmpty(r)
}
