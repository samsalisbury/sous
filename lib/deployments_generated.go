// DO NOT EDIT. Generated with goinline -package=github.com/opentable/sous/util/blueprints/cmap --target-package-name=sous --target-dir=. -w DeployID->DeployID *Deployment->*Deployment

package sous

import (
	"fmt"
	"sync"
)

// Deployments is a wrapper around map[DeployID]*Deployment
// which is safe for concurrent read and write.
type Deployments struct {
	mu *sync.RWMutex
	m  map[DeployID](*Deployment)
}

// MakeDeployments creates a new Deployments with capacity set.
func MakeDeployments(capacity int) Deployments {
	return Deployments{
		mu: &sync.RWMutex{},
		m:  make(map[DeployID](*Deployment), capacity),
	}
}

func (m Deployments) write(f func()) {
	if m.m == nil || m.mu == nil {
		panic("uninitialised Deployments (you should use NewDeployments)")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f()
}

func (m Deployments) read(f func()) {
	if m.m == nil || m.mu == nil {
		panic("uninitialised Deployments (you should use NewDeployments)")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	f()
}

// NewDeploymentsFromMap creates a new Deployments.
// You may optionally pass any number of
// map[DeployID]*Deployments,
// which will be merged key-wise into the new Deployments,
// with keys from the right-most map taking precedence.
func NewDeploymentsFromMap(from ...map[DeployID](*Deployment)) Deployments {
	cm := Deployments{
		mu: &sync.RWMutex{},
		m:  map[DeployID](*Deployment){},
	}
	for _, m := range from {
		for k, v := range m {
			cm.m[k] = v
		}
	}
	return cm
}

// NewDeployments creates a new Deployments.
// You may optionally pass any number of *Deployments,
// which will be added to this map.
func NewDeployments(from ...(*Deployment)) Deployments {
	m := Deployments{
		mu: &sync.RWMutex{},
		m:  map[DeployID](*Deployment){},
	}
	for _, v := range from {
		if !m.Add(v) {
			panic(fmt.Sprintf("conflicting key: %q", v.ID()))
		}
	}
	return m
}

// Get returns (value, true) if k is in the map, or (zero value, false)
// otherwise.
func (m Deployments) Get(key DeployID) (v *Deployment, ok bool) {
	m.read(func() {
		v, ok = m.m[key]
	})
	return
}

// Set sets the value of index k to v.
func (m Deployments) Set(key DeployID, value *Deployment) {
	m.write(func() {
		m.m[key] = value
	})
}

// Filter returns a new Deployments containing only the entries
// where the predicate returns true for the given value.
// A nil predicate is equivalent to calling Clone.
func (m Deployments) Filter(predicate func(*Deployment) bool) Deployments {
	if predicate == nil {
		return m.Clone()
	}
	out := map[DeployID](*Deployment){}
	m.read(func() {
		for k, v := range m.m {
			if predicate(v) {
				out[k] = v
			}
		}
	})
	return NewDeploymentsFromMap(out)
}

// Single returns
// (the single *Deployment satisfying predicate, true),
// if there is exactly one *Deployment satisfying predicate in
// Deployments. Otherwise, returns (zero *Deployment, false).
func (m Deployments) Single(predicate func(*Deployment) bool) (*Deployment, bool) {
	f := m.FilteredSnapshot(predicate)
	if len(f) == 1 {
		for _, v := range f {
			return v, true
		}
	}
	var v (*Deployment)
	return v, false
}

// Any returns
// (a single *Deployment matching predicate, true),
// if there are any *Deployments matching predicate in
// Deployments. Otherwise returns (zero *Deployment, false).
func (m Deployments) Any(predicate func(*Deployment) bool) (*Deployment, bool) {
	f := m.Filter(predicate)
	for _, v := range f.Snapshot() {
		return v, true
	}
	var v (*Deployment)
	return v, false
}

// Clone returns a pairwise copy of Deployments.
func (m Deployments) Clone() Deployments {
	return NewDeploymentsFromMap(m.Snapshot())
}

// Merge returns a new Deployments with
// all entries from this Deployments and the other.
// If any keys in other match keys in this *Deployments,
// keys from other will appear in the returned
// *Deployments.
func (m Deployments) Merge(other Deployments) Deployments {
	return NewDeploymentsFromMap(m.Snapshot(), other.Snapshot())
}

// Add adds a (k, v) pair into a map if it is not already there. Returns true if
// the value was added, false if not.
func (m Deployments) Add(v *Deployment) (ok bool) {
	m.write(func() {
		k := v.ID()
		if _, exists := m.m[k]; exists {
			return
		}
		m.m[k] = v
		ok = true
	})
	return
}

// MustAdd is a wrapper around Add which panics whenever Add returns false.
func (m Deployments) MustAdd(v *Deployment) {
	if !m.Add(v) {
		panic(fmt.Sprintf("item with ID %v already in the graph", v.ID()))
	}
}

// AddAll returns (zero DeployID, true) if all  entries from the passed in
// Deployments have different keys and all are added to this Deployments.
// If any of the keys conflict, nothing will be added to this
// Deployments and AddAll will return the conflicting DeployID and false.
func (m Deployments) AddAll(from Deployments) (conflicting DeployID, success bool) {
	ss := from.Snapshot()
	var exists bool
	m.write(func() {
		for k := range ss {
			if _, exists = m.m[k]; exists {
				conflicting = k
				return
			}
		}
		for k, v := range ss {
			m.m[k] = v
		}
	})
	return conflicting, !exists
}

// Remove value for a key k if present, a no-op otherwise.
func (m Deployments) Remove(key DeployID) {
	m.write(func() {
		delete(m.m, key)
	})
}

// Len returns number of elements in a map.
func (m Deployments) Len() int {
	var l int
	m.read(func() {
		l = len(m.m)
	})
	return l
}

// Keys returns a slice containing all the keys in the map.
func (m Deployments) Keys() []DeployID {
	var keys []DeployID
	m.read(func() {
		keys = make([]DeployID, len(m.m))
		i := 0
		for k := range m.m {
			keys[i] = k
			i++
		}
	})
	return keys
}

// Snapshot returns a moment-in-time copy of the current underlying
// map[DeployID]*Deployment.
func (m Deployments) Snapshot() map[DeployID](*Deployment) {
	var ss map[DeployID](*Deployment)
	m.read(func() {
		ss = make(map[DeployID](*Deployment), len(m.m))
		for k, v := range m.m {
			ss[k] = v
		}
	})
	return ss
}

// FilteredSnapshot returns a moment-in-time filtered copy of the current
// underlying map[DeployID]*Deployment.
// (DeployID, *Deployment) pairs are included
// if they satisfy predicate.
func (m Deployments) FilteredSnapshot(predicate func(*Deployment) bool) map[DeployID](*Deployment) {
	clone := map[DeployID](*Deployment){}
	m.read(func() {
		for k, v := range m.m {
			if predicate(v) {
				clone[k] = v
			}
		}
	})
	return clone
}

// GetAll returns SnapShot (it allows hy to marshal Deployments).
func (m Deployments) GetAll() map[DeployID](*Deployment) {
	return m.Snapshot()
}

// SetAll sets the internal map (it allows hy to unmarshal Deployments).
// Note: SetAll is the only method that is not safe for concurrent access.
func (m *Deployments) SetAll(v map[DeployID](*Deployment)) {
	if m.mu == nil {
		m.mu = &sync.RWMutex{}
	}
	m.m = v
}
