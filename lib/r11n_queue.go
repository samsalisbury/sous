package sous

import "github.com/pborman/uuid"

// R11nQueue is a queue of rectifications.
type R11nQueue struct {
	cap   int
	queue chan *QueuedR11n
	refs  map[R11nID]*QueuedR11n
}

// R11nQueueCapDefault is the default capacity for a new R11nQueue.
const R11nQueueCapDefault = 10

// NewR11nQueue creates a freshly initialised R11nQueue.
func NewR11nQueue(opts ...R11nQueueOpt) *R11nQueue {
	rq := &R11nQueue{
		cap: R11nQueueCapDefault,
	}
	for _, opt := range opts {
		opt(rq)
	}
	return rq.init()
}

// R11nQueueOpt is an option for configuring an R11nQueue.
type R11nQueueOpt func(*R11nQueue)

// R11nQueueCap sets the max capacity of an R11nQueue to the supplied cap.
func R11nQueueCap(cap int) R11nQueueOpt {
	return func(rq *R11nQueue) {
		rq.cap = cap
	}
}

func (rq *R11nQueue) init() *R11nQueue {
	rq.queue = make(chan *QueuedR11n, rq.cap)
	rq.refs = map[R11nID]*QueuedR11n{}
	return rq
}

// QueuedR11n is a queue item wrapping a Rectification with an ID and position.
type QueuedR11n struct {
	ID            R11nID
	Pos           int
	Rectification *Rectification
}

// R11nID is a QueuedR11n identifier.
type R11nID string

// NewR11nID returns a new random R11nID.
func NewR11nID() R11nID {
	return R11nID(uuid.New())
}

// Push adds r to the queue, wrapped in a *QueuedR11n. It returns the wrapper.
// If the push was successful, it returns the wrapper and true, otherwise it
// returns nil and false.
func (rq *R11nQueue) Push(r *Rectification) (*QueuedR11n, bool) {
	if len(rq.queue) == rq.cap {
		return nil, false
	}
	id := NewR11nID()
	qr := &QueuedR11n{
		ID:            id,
		Pos:           len(rq.queue),
		Rectification: r,
	}
	rq.refs[id] = qr
	rq.queue <- qr
	return qr, true
}

// PushIfEmpty adds an item to the queue if it is empty, and returns the wrapper
// added and true if successful. If the queue is not empty, or is full, it
// returns nil, false.
func (rq *R11nQueue) PushIfEmpty(r *Rectification) (*QueuedR11n, bool) {
	if len(rq.queue) != 0 {
		return nil, false
	}
	return rq.Push(r)
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
	for _, r := range rq.refs {
		r.Pos--
	}
	delete(rq.refs, qr.ID)
	return qr, true
}
