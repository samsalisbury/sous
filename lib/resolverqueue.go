package sous

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/opentable/sous/util/logging"
)

// ResolverQueue is a priority queue of SingleDeployResolvers.
// ResolverQueue implements heap.Interface which provides most of the priority
// queue functionality; based on example from
// https://golang.org/src/container/heap/example_pq_test.go
type ResolverQueue struct {
	sr       StateReader
	queue    []*SingleDeployResolver
	deployer Deployer
	registry Registry
	logSet   *logging.LogSet
	*sync.RWMutex
}

// SingleDeployResolver is a resolver configured to resolve only a single
// deployment to a single intended state. After resolving to this state, its
// useful life is over.
type SingleDeployResolver struct {
	Priority, Index int
	DeploymentID    DeploymentID
	Cluster         *Cluster
	Intended        *Deployment
	Resolver        *Resolver
	recorder        *ResolveRecorder
	*sync.RWMutex
}

// Resolve starts resolving this single deployment, waits for completion and
// returns the error. After Resolve is called; Status starts returning proper
// ResolveStatuses rather than "queued at position x" type messages.
func (sdr *SingleDeployResolver) Resolve() error {
	sdr.Lock()
	defer sdr.Unlock()
	sdr.recorder = sdr.Resolver.Begin(NewDeployments(sdr.Intended), Clusters{sdr.Cluster.Name: sdr.Cluster})
	return sdr.recorder.Wait()
}

// Status returns the current ResolveStatus from the internal ResolveRecorder
// (if any), and a textual description of the position in the queue if still
// queued.
func (sdr *SingleDeployResolver) Status() (rs ResolveStatus, desc string) {
	sdr.RLock()
	defer sdr.RUnlock()
	if sdr.recorder == nil {
		return ResolveStatus{}, fmt.Sprintf("queued (number %d in queue)", sdr.Index)
	}
	return sdr.recorder.CurrentStatus(), "front of queue (working on this now)"
}

// NewResolverQueue creates a new ResolverQueue and populates it with a
// separate Resolver for each Deployment from sr filtered by rf.
func NewResolverQueue(sr StateReader, d Deployer, r Registry, rf *ResolveFilter, ls logging.LogSink) *ResolverQueue {
	return &ResolverQueue{
		deployer: d,
		registry: r,
		RWMutex:  &sync.RWMutex{},
	}
}

// Start begins reading state and rectifying each deployment.
func (r *ResolverQueue) Start() error {
	r.Lock()
	defer r.Unlock()
	intended, err := r.intended()
	if err != nil {
		return err
	}
	r.queue = make([]*SingleDeployResolver, intended.Len())
	var i int
	for id, d := range intended.Snapshot() {
		filter := SingleDeploymentResolveFilter(id)
		resolver := NewResolver(r.deployer, r.registry, filter, r.logSet)
		r.queue[i] = &SingleDeployResolver{
			Priority:     100,
			DeploymentID: id,
			Intended:     d,
			Resolver:     resolver,
			RWMutex:      &sync.RWMutex{},
		}
	}
	go r.mainLoop()
	return nil
}

func (r *ResolverQueue) mainLoop() {
	for {
		sdr, ok := r.Next()
		if !ok {
			timeout := 10 * time.Second
			r.logSet.Warnf("deployer queue is empty, checking again in %s", timeout)
			time.Sleep(timeout)
			continue
		}
		if err := sdr.Resolve(); err != nil {
			// TODO: Structured log with DeployID fields for filtering.
			r.logSet.Warnf("Resolving %s failed: %s", sdr.DeploymentID, err)
		}
	}
}

func (r *ResolverQueue) intended() (Deployments, error) {
	if state, err := r.sr.ReadState(); err != nil {
		return Deployments{}, err
	} else if ds, err := state.Deployments(); err != nil {
		return Deployments{}, err
	} else {
		return ds, nil
	}
}

// Update modifies the priority of the resolver with DeploymentID id.
func (r *ResolverQueue) Update(id DeploymentID, priority int) {
	for _, resolver := range r.queue {
		if resolver.DeploymentID == id {
			resolver.Priority = priority
			heap.Fix(r, resolver.Index)
			return
		}
	}
}

// Next returns the highest priority resolver.
func (r *ResolverQueue) Next() (*SingleDeployResolver, bool) {
	r.Lock()
	defer r.Unlock()
	if r.Len() == 0 {
		return nil, false
	}
	return r.Pop().(*SingleDeployResolver), true
}

// Len for heap.Interface.
func (r *ResolverQueue) Len() int {
	return len(r.queue)
}

// Less for heap.Interface.
func (r *ResolverQueue) Less(i int, j int) bool {
	// We want Pop to give us the highest, not lowest,
	// priority so we use greater than here.
	return r.queue[i].Priority > r.queue[j].Priority
}

// Swap for heap.Interface.
func (r *ResolverQueue) Swap(i int, j int) {
	r.queue[i], r.queue[j] = r.queue[j], r.queue[i]
	r.queue[i].Index = i
	r.queue[j].Index = j
}

// Push for heap.Interface.
func (r *ResolverQueue) Push(x interface{}) {
	n := len(r.queue)
	item := x.(*SingleDeployResolver)
	item.Index = n
	r.queue = append(r.queue, item)
}

// Pop for heap.Interface.
func (r *ResolverQueue) Pop() interface{} {
	old := r.queue
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	r.queue = old[0 : n-1]
	return item
}
