package cmap

import (
	"fmt"
	"sync"
)

// CMap is a wrapper around map[CMKey]Value
// which is safe for concurrent read and write.
type CMap struct {
	mu *sync.RWMutex
	m  map[CMKey]Value
}

// CMKey is the map key type.
type CMKey string

// Value is the map value type.
type Value string

// MakeCMap creates a new CMap with capacity set.
func MakeCMap(capacity int) CMap {
	return CMap{
		mu: &sync.RWMutex{},
		m:  make(map[CMKey]Value, capacity),
	}
}

// NewCMapFromMap creates a new CMap.
// You may optionally pass any number of
// map[CMKey]Values,
// which will be merged key-wise into the new CMap,
// with keys from the right-most map taking precedence.
func NewCMapFromMap(from ...map[CMKey]Value) CMap {
	cm := CMap{
		mu: &sync.RWMutex{},
		m:  map[CMKey]Value{},
	}
	for _, m := range from {
		for k, v := range m {
			cm.m[k] = v
		}
	}
	return cm
}

// NewCMap creates a new CMap.
// You may optionally pass any number of Values,
// which will be added to this map.
func NewCMap(from ...Value) CMap {
	m := CMap{
		mu: &sync.RWMutex{},
		m:  map[CMKey]Value{},
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
func (m *CMap) Get(key CMKey) (Value, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.m[key]
	return v, ok
}

// Set sets the value of index k to v.
func (m *CMap) Set(key CMKey, value Value) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[key] = value
}

// Filter returns a new CMap containing only the entries
// where the predicate returns true for the given value.
// A nil predicate is equivalent to calling Clone.
func (m *CMap) Filter(predicate func(Value) bool) CMap {
	if predicate == nil {
		return m.Clone()
	}
	out := map[CMKey]Value{}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.m {
		if predicate(v) {
			out[k] = v
		}
	}
	return NewCMapFromMap(out)
}

// Single returns
// (the single Value satisfying predicate, true),
// if there is exactly one Value satisfying predicate in
// CMap. Otherwise, returns (zero Value, false).
func (m *CMap) Single(predicate func(Value) bool) (Value, bool) {
	f := m.FilteredSnapshot(predicate)
	if len(f) == 1 {
		for _, v := range f {
			return v, true
		}
	}
	var v Value
	return v, false
}

// Any returns
// (a single Value matching predicate, true),
// if there are any Values matching predicate in
// CMap. Otherwise returns (zero Value, false).
func (m *CMap) Any(predicate func(Value) bool) (Value, bool) {
	f := m.Filter(predicate)
	for _, v := range f.Snapshot() {
		return v, true
	}
	var v Value
	return v, false
}

// Clone returns a pairwise copy of CMap.
func (m *CMap) Clone() CMap {
	return NewCMapFromMap(m.Snapshot())
}

// Merge returns a new CMap with
// all entries from this CMap and the other.
// If any keys in other match keys in this *CMap,
// keys from other will appear in the returned
// *CMap.
func (m *CMap) Merge(other CMap) CMap {
	return NewCMapFromMap(m.Snapshot(), other.Snapshot())
}

// Add adds a (k, v) pair into a map if it is not already there. Returns true if
// the value was added, false if not.
func (m *CMap) Add(v Value) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := v.ID()
	if _, exists := m.m[k]; exists {
		return false
	}
	m.m[k] = v
	return true
}

// MustAdd is a wrapper around Add which panics whenever Add returns false.
func (m *CMap) MustAdd(v Value) {
	if !m.Add(v) {
		panic(fmt.Sprintf("item with ID %v already in the graph", v.ID()))
	}
}

// AddAll returns (zero CMKey, true) if all  entries from the passed in
// CMap have different keys and all are added to this CMap.
// If any of the keys conflict, nothing will be added to this
// CMap and AddAll will return the conflicting CMKey and false.
func (m *CMap) AddAll(from CMap) (conflicting CMKey, success bool) {
	ss := from.Snapshot()
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range ss {
		if _, exists := m.m[k]; exists {
			m.mu.RUnlock()
			return k, false
		}
	}
	for k, v := range ss {
		m.m[k] = v
	}
	return conflicting, true
}

// Remove value for a key k if present, a no-op otherwise.
func (m *CMap) Remove(key CMKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
}

// Len returns number of elements in a map.
func (m *CMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}

// Keys returns a slice containing all the keys in the map.
func (m *CMap) Keys() []CMKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]CMKey, len(m.m))
	i := 0
	for k := range m.m {
		keys[i] = k
		i++
	}
	return keys
}

// Snapshot returns a moment-in-time copy of the current underlying
// map[CMKey]Value.
func (m *CMap) Snapshot() map[CMKey]Value {
	m.mu.RLock()
	defer m.mu.RUnlock()
	clone := make(map[CMKey]Value, len(m.m))
	for k, v := range m.m {
		clone[k] = v
	}
	return clone
}

// FilteredSnapshot returns a moment-in-time filtered copy of the current
// underlying map[CMKey]Value.
// (CMKey, Value) pairs are included
// if they satisfy predicate.
func (m *CMap) FilteredSnapshot(predicate func(Value) bool) map[CMKey]Value {
	m.mu.RLock()
	defer m.mu.RUnlock()
	clone := map[CMKey]Value{}
	for k, v := range m.m {
		if predicate(v) {
			clone[k] = v
		}
	}
	return clone
}
