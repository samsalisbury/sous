package sous

import (
	"sync"

	"github.com/pborman/uuid"
)

// R11nQueue is a queue of rectifications.
type R11nQueue struct {
	cap   int
	queue chan *QueuedR11n
	refs  map[R11nID]*QueuedR11n
	sync.Mutex
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

// R11nQueueStartWithHandler starts processing the queue using the supplied
// handler.
func R11nQueueStartWithHandler(handler func(*QueuedR11n) DiffResolution) R11nQueueOpt {
	return func(rq *R11nQueue) {
		rq.Start(handler)
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
	done          chan struct{}
}

// R11nID is a QueuedR11n identifier.
type R11nID string

// NewR11nID returns a new random R11nID.
func NewR11nID() R11nID {
	return R11nID(uuid.New())
}

// Start starts applying handler to each item on the queue in order.
func (rq *R11nQueue) Start(handler func(*QueuedR11n) DiffResolution) <-chan DiffResolution {
	results := make(chan DiffResolution, 100)
	go func() {
		for {
			qr := rq.next()
			results <- handler(qr)
			close(qr.done)
			rq.Lock()
			delete(rq.refs, qr.ID)
			rq.Unlock()
		}
	}()
	return results
}

// Wait waits for a particular rectification to be processed then returns its
// result. If that rectification is not in this queue, it immediately returns a
// zero DiffResolution and false.
func (rq *R11nQueue) Wait(id R11nID) (DiffResolution, bool) {
	rq.Lock()
	qr, ok := rq.refs[id]
	rq.Unlock()
	if ok {
		<-qr.done
		return qr.Rectification.Resolution, true
	}
	return DiffResolution{}, false
}

// Push adds r to the queue, wrapped in a *QueuedR11n. It returns the wrapper.
// If the push was successful, it returns the wrapper and true, otherwise it
// returns nil and false.
func (rq *R11nQueue) Push(r *Rectification) (*QueuedR11n, bool) {
	rq.Lock()
	defer rq.Unlock()
	if len(rq.queue) == rq.cap {
		return nil, false
	}
	return rq.internalPush(r), true
}

// internalPush assumes rq is already locked.
func (rq *R11nQueue) internalPush(r *Rectification) *QueuedR11n {
	id := NewR11nID()
	qr := &QueuedR11n{
		ID:            id,
		Pos:           len(rq.queue),
		Rectification: r,
		done:          make(chan struct{}),
	}
	rq.refs[id] = qr
	rq.queue <- qr
	return qr
}

// PushIfEmpty adds an item to the queue if it is empty, and returns the wrapper
// added and true if successful. If the queue is not empty, or is full, it
// returns nil, false.
func (rq *R11nQueue) PushIfEmpty(r *Rectification) (*QueuedR11n, bool) {
	rq.Lock()
	defer rq.Unlock()
	// We look at refs since we only delete the ref after handling has happened.
	// If we are busy handling a r11n, then we consider the queue non-empty.
	if len(rq.refs) != 0 {
		return nil, false
	}
	return rq.internalPush(r), true
}

// Len returns the current number of items in the queue.
func (rq *R11nQueue) Len() int {
	return len(rq.queue)
}

// pop removes the item at the front of the queue and returns it plus true.
// It returns nil and false if there are no items in the queue.
func (rq *R11nQueue) pop() (*QueuedR11n, bool) {
	rq.Lock()
	defer rq.Unlock()
	if len(rq.queue) == 0 {
		return nil, false
	}
	qr := <-rq.queue
	rq.handlePopped(qr.ID)
	return qr, true
}

// next is similar to pop but waits until there is something on the queue to
// return and then returns it.
func (rq *R11nQueue) next() *QueuedR11n {
	if qr, ok := rq.pop(); ok {
		return qr
	}
	qr := <-rq.queue
	rq.Lock()
	defer rq.Unlock()
	rq.handlePopped(qr.ID)
	return qr
}

// handlePopped assumes rq is locked.
func (rq *R11nQueue) handlePopped(id R11nID) {
	for _, r := range rq.refs {
		r.Pos--
	}
}
