// DO NOT EDIT. Generated with goinline -package=github.com/opentable/sous/util/blueprints/cmap --target-package-name=sous --target-dir=. -w DeployID->DeployID *DeployState->*DeployState

package sous

import (
	"fmt"
	"sync"
)

// DeployStates is a wrapper around map[DeployID]*DeployState
// which is safe for concurrent read and write.
type DeployStates struct {
	mu *sync.RWMutex
	m  map[DeploymentID](*DeployState)
}

// MakeDeployStates creates a new DeployStates with capacity set.
func MakeDeployStates(capacity int) DeployStates {
	return DeployStates{
		mu: &sync.RWMutex{},
		m:  make(map[DeploymentID](*DeployState), capacity),
	}
}

func (m DeployStates) write(f func()) {
	if m.m == nil || m.mu == nil {
		panic("uninitialised DeployStates (you should use NewDeployStates)")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f()
}

func (m DeployStates) read(f func()) {
	if m.m == nil || m.mu == nil {
		panic("uninitialised DeployStates (you should use NewDeployStates)")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	f()
}

// NewDeployStatesFromMap creates a new DeployStates.
// You may optionally pass any number of
// map[DeployID]*DeployStates,
// which will be merged key-wise into the new DeployStates,
// with keys from the right-most map taking precedence.
func NewDeployStatesFromMap(from ...map[DeploymentID](*DeployState)) DeployStates {
	cm := DeployStates{
		mu: &sync.RWMutex{},
		m:  map[DeploymentID](*DeployState){},
	}
	for _, m := range from {
		for k, v := range m {
			cm.m[k] = v
		}
	}
	return cm
}

// NewDeployStates creates a new DeployStates.
// You may optionally pass any number of *DeployStates,
// which will be added to this map.
func NewDeployStates(from ...(*DeployState)) DeployStates {
	m := DeployStates{
		mu: &sync.RWMutex{},
		m:  map[DeploymentID](*DeployState){},
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
func (m DeployStates) Get(key DeploymentID) (v *DeployState, ok bool) {
	m.read(func() {
		v, ok = m.m[key]
	})
	return
}

// Set sets the value of index k to v.
func (m DeployStates) Set(key DeploymentID, value *DeployState) {
	m.write(func() {
		m.m[key] = value
	})
}

// Filter returns a new DeployStates containing only the entries
// where the predicate returns true for the given value.
// A nil predicate is equivalent to calling Clone.
func (m DeployStates) Filter(predicate func(*DeployState) bool) DeployStates {
	if predicate == nil {
		return m.Clone()
	}
	out := map[DeploymentID](*DeployState){}
	m.read(func() {
		for k, v := range m.m {
			if predicate(v) {
				out[k] = v
			}
		}
	})
	return NewDeployStatesFromMap(out)
}

// Single returns
// (the single *DeployState satisfying predicate, true),
// if there is exactly one *DeployState satisfying predicate in
// DeployStates. Otherwise, returns (zero *DeployState, false).
func (m DeployStates) Single(predicate func(*DeployState) bool) (*DeployState, bool) {
	f := m.FilteredSnapshot(predicate)
	if len(f) == 1 {
		for _, v := range f {
			return v, true
		}
	}
	var v (*DeployState)
	return v, false
}

// Any returns
// (a single *DeployState matching predicate, true),
// if there are any *DeployStates matching predicate in
// DeployStates. Otherwise returns (zero *DeployState, false).
func (m DeployStates) Any(predicate func(*DeployState) bool) (*DeployState, bool) {
	f := m.Filter(predicate)
	for _, v := range f.Snapshot() {
		return v, true
	}
	var v (*DeployState)
	return v, false
}

// Clone returns a pairwise copy of DeployStates.
func (m DeployStates) Clone() DeployStates {
	c := NewDeployStates()
	for _, v := range m.Snapshot() {
		c.Add(v.Clone())
	}
	return c
}

// Merge returns a new DeployStates with
// all entries from this DeployStates and the other.
// If any keys in other match keys in this *DeployStates,
// keys from other will appear in the returned
// *DeployStates.
func (m DeployStates) Merge(other DeployStates) DeployStates {
	return NewDeployStatesFromMap(m.Snapshot(), other.Snapshot())
}

// Add adds a (k, v) pair into a map if it is not already there. Returns true if
// the value was added, false if not.
func (m DeployStates) Add(v *DeployState) (ok bool) {
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
func (m DeployStates) MustAdd(v *DeployState) {
	if !m.Add(v) {
		panic(fmt.Sprintf("item with ID %v already in the graph", v.ID()))
	}
}

// AddAll returns (zero DeployID, true) if all  entries from the passed in
// DeployStates have different keys and all are added to this DeployStates.
// If any of the keys conflict, nothing will be added to this
// DeployStates and AddAll will return the conflicting DeployID and false.
func (m DeployStates) AddAll(from DeployStates) (conflicting DeploymentID, success bool) {
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
func (m DeployStates) Remove(key DeploymentID) {
	m.write(func() {
		delete(m.m, key)
	})
}

// Len returns number of elements in a map.
func (m DeployStates) Len() int {
	var l int
	m.read(func() {
		l = len(m.m)
	})
	return l
}

// Keys returns a slice containing all the keys in the map.
func (m DeployStates) Keys() []DeploymentID {
	var keys []DeploymentID
	m.read(func() {
		keys = make([]DeploymentID, len(m.m))
		i := 0
		for k := range m.m {
			keys[i] = k
			i++
		}
	})
	return keys
}

// Snapshot returns a moment-in-time copy of the current underlying
// map[DeployID]*DeployState.
func (m DeployStates) Snapshot() map[DeploymentID](*DeployState) {
	var ss map[DeploymentID](*DeployState)
	m.read(func() {
		ss = make(map[DeploymentID](*DeployState), len(m.m))
		for k, v := range m.m {
			ss[k] = v
		}
	})
	return ss
}

// FilteredSnapshot returns a moment-in-time filtered copy of the current
// underlying map[DeployID]*DeployState.
// (DeployID, *DeployState) pairs are included
// if they satisfy predicate.
func (m DeployStates) FilteredSnapshot(predicate func(*DeployState) bool) map[DeploymentID](*DeployState) {
	clone := map[DeploymentID](*DeployState){}
	m.read(func() {
		for k, v := range m.m {
			if predicate(v) {
				clone[k] = v
			}
		}
	})
	return clone
}

// GetAll returns SnapShot (it allows hy to marshal DeployStates).
func (m DeployStates) GetAll() map[DeploymentID](*DeployState) {
	return m.Snapshot()
}

// SetAll sets the internal map (it allows hy to unmarshal DeployStates).
// Note: SetAll is the only method that is not safe for concurrent access.
func (m *DeployStates) SetAll(v map[DeploymentID](*DeployState)) {
	if m.mu == nil {
		m.mu = &sync.RWMutex{}
	}
	m.m = v
}
