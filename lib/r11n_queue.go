package sous

import (
	"container/ring"
	"sort"
	"sync"

	"github.com/pborman/uuid"
)

// MaxRefsPerR11nQueue is the maximum number of rectifications to cache in memory.
const MaxRefsPerR11nQueue = 100

// R11nQueue is a queue of rectifications.
type R11nQueue struct {
	cap           int
	queue         chan *QueuedR11n
	refs, allRefs map[R11nID]*QueuedR11n
	fifoRefs      *ring.Ring
	handler       func(*QueuedR11n) DiffResolution
	start         bool
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
	rq.init()
	if rq.start {
		rq.Start(rq.handler)
	}
	return rq
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
		rq.handler = func(qr *QueuedR11n) DiffResolution {
			dr := handler(qr)
			// TODO SS:
			// This oddity ensures the resolution on the queued rectification
			// matches that returned by the handler. This is only really
			// important in testing where we don't want to run rectifications
			// just to test the queue. However I would rather clean up the
			// implementation to remove the need for this.
			qr.Rectification.Resolution = dr
			return dr
		}
		rq.start = true
	}
}

// Snapshot returns a slice of items to be processed in the queue ordered by
// their queue position. It includes the item being worked on at the head of the
// queue.
func (rq *R11nQueue) Snapshot() []QueuedR11n {
	rq.Lock()
	defer rq.Unlock()
	var snapshot []QueuedR11n
	for _, qr := range rq.refs {
		snapshot = append(snapshot, *qr)
	}
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].Pos < snapshot[j].Pos
	})
	return snapshot
}

// ByID returns the queued rectification matching ID and true if it exists, nil
// and false otherwise.
func (rq *R11nQueue) ByID(id R11nID) (*QueuedR11n, bool) {
	rq.Lock()
	defer rq.Unlock()
	qr, ok := rq.refs[id]
	return qr, ok
}

func (rq *R11nQueue) init() *R11nQueue {
	rq.Lock()
	defer rq.Unlock()
	rq.queue = make(chan *QueuedR11n, rq.cap)
	rq.refs = map[R11nID]*QueuedR11n{}
	rq.allRefs = map[R11nID]*QueuedR11n{}
	rq.fifoRefs = ring.New(MaxRefsPerR11nQueue)
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
func (rq *R11nQueue) Start(handler func(*QueuedR11n) DiffResolution) {
	rq.Lock()
	defer rq.Unlock()
	go func() {
		for {
			qr := rq.next()
			handler(qr)
			rq.Lock()
			close(qr.done)
			delete(rq.refs, qr.ID)
			rq.Unlock()
		}
	}()
}

// Wait waits for a particular rectification to be processed then returns its
// result. If that rectification is not in this queue, it immediately returns a
// zero DiffResolution and false.
func (rq *R11nQueue) Wait(id R11nID) (DiffResolution, bool) {
	rq.Lock()
	qr, ok := rq.allRefs[id]
	rq.Unlock()
	if !ok {
		return DiffResolution{}, false
	}
	<-qr.done
	return qr.Rectification.Resolution, true
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
	rq.allRefs[id] = qr
	rq.fifoRefs = rq.fifoRefs.Next()
	if rq.fifoRefs.Value != nil {
		idToDelete := rq.fifoRefs.Value.(R11nID)
		delete(rq.allRefs, idToDelete)
	}
	rq.fifoRefs.Value = id
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

// next waits until there is something on the queue to
// return and then returns it.
func (rq *R11nQueue) next() *QueuedR11n {
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
