package sous

import "github.com/pborman/uuid"

// R11nQueue is a queue of rectifications.
type R11nQueue struct {
	queue chan *QueuedR11n
	refs  map[R11nID]*QueuedR11n
}

// QueuedR11n is a queue item wrapping a Rectification with an ID and position.
type QueuedR11n struct {
	ID            R11nID
	Pos           int
	Rectification Rectification
}

// R11nID is a QueuedR11n identifier.
type R11nID string

// NewR11nID returns a new random R11nID.
func NewR11nID() R11nID {
	return R11nID(uuid.New())
}

// NewR11nQueue creates a freshly initialised R11nQueue.
func NewR11nQueue() *R11nQueue {
	return &R11nQueue{
		queue: make(chan *QueuedR11n, 10),
		refs:  map[R11nID]*QueuedR11n{},
	}
}

// Push adds r to the queue, wrapped in a *QueuedR11n. It returns the wrapper.
func (rq *R11nQueue) Push(r *Rectification) *QueuedR11n {
	id := NewR11nID()
	qr := &QueuedR11n{
		ID:  id,
		Pos: len(rq.queue),
	}
	rq.refs[id] = qr
	rq.queue <- qr
	return qr
}

// PushIfEmpty adds an item to the queue if it is empty, and returns the wrapper
// added and true if successful. If the queue is not empty, it returns
// nil, false.
func (rq *R11nQueue) PushIfEmpty(r *Rectification) (*QueuedR11n, bool) {
	if len(rq.queue) != 0 {
		return nil, false
	}
	return rq.Push(r), true
}

// Len returns the current number of items in the queue.
func (rq *R11nQueue) Len() int {
	return len(rq.queue)
}

// Pop removes the item at the front of the queue and returns it plus true.
// It returns nil and false if there are no items in the queue.
func (rq *R11nQueue) Pop() (*QueuedR11n, bool) {
	if len(rq.queue) == 0 {
		return nil, false
	}
	qr := <-rq.queue
	delete(rq.refs, qr.ID)
	for _, r := range rq.refs {
		r.Pos--
	}
	return qr, true
}
