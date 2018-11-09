package psyringe

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
)

type itm map[reflect.Type]*injectionType

type injectionTypes struct {
	m itm
	sync.RWMutex
}

type injectionType struct {
	Ctor               *ctor
	Value              reflect.Value
	DebugAddedLocation string
}

func newInjectionTypes() *injectionTypes {
	return &injectionTypes{m: itm{}}
}

// Keys returns a sorted slice of the reflect.Type keys of this collection.
func (its *injectionTypes) Keys() []reflect.Type {
	types := make([]reflect.Type, len(its.m))
	i := 0
	its.RLock()
	defer its.RUnlock()
	for t := range its.m {
		types[i] = t
		i++
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name() < types[j].Name()
	})
	return types
}

func (its *injectionTypes) Contains(t reflect.Type) bool {
	its.RLock()
	defer its.RUnlock()
	_, ok := its.m[t]
	return ok
}

func (its *injectionTypes) Add(t reflect.Type, it *injectionType) error {
	if its.Contains(t) {
		return fmt.Errorf("type %s already registered", t)
	}
	its.Lock()
	defer its.Unlock()
	its.m[t] = it
	return nil
}

func (its *injectionTypes) Clone() *injectionTypes {
	its.RLock()
	defer its.RUnlock()
	m := make(itm, len(its.m))
	for t, it := range its.m {
		m[t] = it.Clone()
	}
	return &injectionTypes{m: m}
}

func (it *injectionType) Clone() *injectionType {
	clone := *it
	if it.Ctor != nil {
		clone.Ctor = it.Ctor.clone()
	}
	if (it.Value != reflect.Value{}) {
		clone.Value = it.Value
	}
	return &clone
}

func (its *injectionTypes) GetOrNil(key reflect.Type) *injectionType {
	its.RLock()
	defer its.RUnlock()
	return its.m[key]
}

func (its *injectionTypes) Get(key reflect.Type) (*injectionType, bool) {
	its.RLock()
	defer its.RUnlock()
	i, ok := its.m[key]
	return i, ok
}

func (its *injectionTypes) Delete(key reflect.Type) {
	its.Lock()
	defer its.Unlock()
	delete(its.m, key)
}

// Where filters injectionTypes by the predicate.
func (its *injectionTypes) Where(predicate func(reflect.Type, *injectionType) bool) *injectionTypes {
	filtered := itm{}
	its.RLock()
	defer its.RUnlock()
	for t, it := range its.m {
		if predicate(t, it) {
			filtered[t] = it
		}
	}
	return &injectionTypes{m: filtered}
}

// AddedAsCtors filters injection types to those representing constructors (note: they
// may have already been called and also have a value if so.
func (its *injectionTypes) AddedAsCtors() *injectionTypes {
	return its.Where(func(_ reflect.Type, it *injectionType) bool {
		return it.Ctor != nil
	})
}

// AddedAsValues filters injection types to those representing values (note: this
// excludes values created internally by constructors.
func (its *injectionTypes) AddedAsValues() *injectionTypes {
	return its.Where(func(_ reflect.Type, it *injectionType) bool {
		return it.Ctor == nil
	})
}

func (its *injectionTypes) WithRealisedValues() *injectionTypes {
	return its.Where(func(_ reflect.Type, it *injectionType) bool {
		return it.Value.IsValid()
	})
}
