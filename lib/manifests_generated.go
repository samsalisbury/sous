// DO NOT EDIT. Generated with goinline -package=github.com/opentable/sous/util/blueprints/cmap --target-package-name=sous --target-dir=. -w ManifestID->ManifestID *Manifest->*Manifest

package sous

import (
	"fmt"
	"sync"
)

// Manifests is a wrapper around map[ManifestID]*Manifest
// which is safe for concurrent read and write.
type Manifests struct {
	mu *sync.RWMutex
	m  map[ManifestID](*Manifest)
}

// MakeManifests creates a new Manifests with capacity set.
func MakeManifests(capacity int) Manifests {
	return Manifests{
		mu: &sync.RWMutex{},
		m:  make(map[ManifestID](*Manifest), capacity),
	}
}

func (m Manifests) write(f func()) {
	if m.m == nil || m.mu == nil {
		panic("uninitialised Manifests (you should use NewManifests)")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f()
}

func (m Manifests) read(f func()) {
	if m.m == nil || m.mu == nil {
		panic("uninitialised Manifests (you should use NewManifests)")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	f()
}

// NewManifestsFromMap creates a new Manifests.
// You may optionally pass any number of
// map[ManifestID]*Manifests,
// which will be merged key-wise into the new Manifests,
// with keys from the right-most map taking precedence.
func NewManifestsFromMap(from ...map[ManifestID](*Manifest)) Manifests {
	cm := Manifests{
		mu: &sync.RWMutex{},
		m:  map[ManifestID](*Manifest){},
	}
	for _, m := range from {
		for k, v := range m {
			cm.m[k] = v
		}
	}
	return cm
}

// NewManifests creates a new Manifests.
// You may optionally pass any number of *Manifests,
// which will be added to this map.
func NewManifests(from ...(*Manifest)) Manifests {
	m := Manifests{
		mu: &sync.RWMutex{},
		m:  map[ManifestID](*Manifest){},
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
func (m Manifests) Get(key ManifestID) (v *Manifest, ok bool) {
	m.read(func() {
		v, ok = m.m[key]
	})
	return
}

// Set sets the value of index k to v.
func (m Manifests) Set(key ManifestID, value *Manifest) {
	m.write(func() {
		m.m[key] = value
	})
}

// Filter returns a new Manifests containing only the entries
// where the predicate returns true for the given value.
// A nil predicate is equivalent to calling Clone.
func (m Manifests) Filter(predicate func(*Manifest) bool) Manifests {
	if predicate == nil {
		return m.Clone()
	}
	out := map[ManifestID](*Manifest){}
	m.read(func() {
		for k, v := range m.m {
			if predicate(v) {
				out[k] = v
			}
		}
	})
	return NewManifestsFromMap(out)
}

// Single returns
// (the single *Manifest satisfying predicate, true),
// if there is exactly one *Manifest satisfying predicate in
// Manifests. Otherwise, returns (zero *Manifest, false).
func (m Manifests) Single(predicate func(*Manifest) bool) (*Manifest, bool) {
	f := m.FilteredSnapshot(predicate)
	if len(f) == 1 {
		for _, v := range f {
			return v, true
		}
	}
	var v (*Manifest)
	return v, false
}

// Any returns
// (a single *Manifest matching predicate, true),
// if there are any *Manifests matching predicate in
// Manifests. Otherwise returns (zero *Manifest, false).
func (m Manifests) Any(predicate func(*Manifest) bool) (*Manifest, bool) {
	f := m.Filter(predicate)
	for _, v := range f.Snapshot() {
		return v, true
	}
	var v (*Manifest)
	return v, false
}

// Clone returns a pairwise copy of Manifests.
func (m Manifests) Clone() Manifests {
	c := NewManifests()
	for _, v := range m.Snapshot() {
		c.Add(v.Clone())
	}
	return c
}

// Merge returns a new Manifests with
// all entries from this Manifests and the other.
// If any keys in other match keys in this *Manifests,
// keys from other will appear in the returned
// *Manifests.
func (m Manifests) Merge(other Manifests) Manifests {
	return NewManifestsFromMap(m.Snapshot(), other.Snapshot())
}

// Add adds a (k, v) pair into a map if it is not already there. Returns true if
// the value was added, false if not.
func (m Manifests) Add(v *Manifest) (ok bool) {
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
func (m Manifests) MustAdd(v *Manifest) {
	if !m.Add(v) {
		panic(fmt.Sprintf("item with ID %v already in the graph", v.ID()))
	}
}

// AddAll returns (zero ManifestID, true) if all  entries from the passed in
// Manifests have different keys and all are added to this Manifests.
// If any of the keys conflict, nothing will be added to this
// Manifests and AddAll will return the conflicting ManifestID and false.
func (m Manifests) AddAll(from Manifests) (conflicting ManifestID, success bool) {
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
func (m Manifests) Remove(key ManifestID) {
	m.write(func() {
		delete(m.m, key)
	})
}

// Len returns number of elements in a map.
func (m Manifests) Len() int {
	var l int
	m.read(func() {
		l = len(m.m)
	})
	return l
}

// Keys returns a slice containing all the keys in the map.
func (m Manifests) Keys() []ManifestID {
	var keys []ManifestID
	m.read(func() {
		keys = make([]ManifestID, len(m.m))
		i := 0
		for k := range m.m {
			keys[i] = k
			i++
		}
	})
	return keys
}

// Snapshot returns a moment-in-time copy of the current underlying
// map[ManifestID]*Manifest.
func (m Manifests) Snapshot() map[ManifestID](*Manifest) {
	var ss map[ManifestID](*Manifest)
	m.read(func() {
		ss = make(map[ManifestID](*Manifest), len(m.m))
		for k, v := range m.m {
			ss[k] = v
		}
	})
	return ss
}

// FilteredSnapshot returns a moment-in-time filtered copy of the current
// underlying map[ManifestID]*Manifest.
// (ManifestID, *Manifest) pairs are included
// if they satisfy predicate.
func (m Manifests) FilteredSnapshot(predicate func(*Manifest) bool) map[ManifestID](*Manifest) {
	clone := map[ManifestID](*Manifest){}
	m.read(func() {
		for k, v := range m.m {
			if predicate(v) {
				clone[k] = v
			}
		}
	})
	return clone
}

// GetAll returns SnapShot (it allows hy to marshal Manifests).
func (m Manifests) GetAll() map[ManifestID](*Manifest) {
	return m.Snapshot()
}

// SetAll sets the internal map (it allows hy to unmarshal Manifests).
// Note: SetAll is the only method that is not safe for concurrent access.
func (m *Manifests) SetAll(v map[ManifestID](*Manifest)) {
	if m.mu == nil {
		m.mu = &sync.RWMutex{}
	}
	m.m = v
}
