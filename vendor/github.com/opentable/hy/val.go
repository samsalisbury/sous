package hy

import (
	"reflect"
)

// Val wraps a reflect.Value so other code does not need to be aware of pointer
// wrapping and unwrapping.
type Val struct {
	Base *NodeBase
	// Ptr is the underlying pointer value.
	Ptr reflect.Value
	// Key is the associated key for this value. May be invalid.
	Key reflect.Value
	// IsPtr indicates whether the final version of this value should be a
	// pointer.
	IsPtr bool
}

// NewFreeValFrom creates a new free value (not attached to any node).
func NewFreeValFrom(v reflect.Value) Val {
	if v.Kind() == reflect.Ptr {
		return Val{Ptr: v, IsPtr: true}
	}
	if v.CanAddr() {
		return Val{Ptr: v.Addr()}
	}
	ptr := reflect.New(v.Type())
	ptr.Elem().Set(v)
	return Val{Ptr: ptr}
}

// Final returns the final reflect.Value.
func (v Val) Final() reflect.Value {
	if v.IsPtr {
		return v.Ptr
	}
	return v.Ptr.Elem()
}

// IsZero means "is zero or nil or invalid".
func (v Val) IsZero() bool {
	return v.Ptr.IsNil() ||
		!v.Ptr.Elem().IsValid() ||
		reflect.DeepEqual(v.Ptr.Elem().Interface(), v.Base.Zero)
}

// ShouldWrite returns true if this value is not zero or if it does have a key.
// Zero unkeyed values are never written.
func (v Val) ShouldWrite() bool {
	return !v.IsZero() || v.Base.HasKey
}

// SetField sets a field on this struct value.
// It panics if this value is not a struct.
func (v Val) SetField(name string, val Val) {
	v.Ptr.Elem().FieldByName(name).Set(val.Final())
}

// GetField gets a field value by name.
// It panics if this value is not a struct.
func (v Val) GetField(name string) reflect.Value {
	return v.Ptr.Elem().FieldByName(name)
}

// Method wraps MethodByName.
func (v Val) Method(name string) reflect.Value {
	if _, ok := v.Ptr.Type().MethodByName(name); ok {
		return v.Ptr.MethodByName(name)
	}
	return v.Ptr.Elem().MethodByName(name)
}

// SetMapElement sets a map element using key and value from val.
// It panics if this value is not a map with corresponding key and value types.
func (v Val) SetMapElement(val Val) {
	v.Ptr.Elem().SetMapIndex(val.Key, val.Final())
}

// MapElements returns a slice of Val representing key, value pairs from this
// map.
// It panics if v does not represent a map.
func (v Val) MapElements(elemNode Node) []Val {
	m := v.Ptr.Elem()
	vals := make([]Val, m.Len())
	for i, key := range m.MapKeys() {
		vals[i] = elemNode.NewKeyedValFrom(key, m.MapIndex(key))
	}
	return vals
}

// Append appends a val to this slice.
// It panics if v is not a slice.
func (v Val) Append(val Val) {
	reflect.Append(v.Ptr.Elem(), val.Final())
}

// Interface selects the pointer or non-pointer version of this val that
// satisfies the filter function. It returns nil, false if none pass the filter.
func (v Val) Interface(filter func(interface{}) bool) (interface{}, bool) {
	ptrInterface := v.Ptr.Interface()
	if filter(ptrInterface) {
		return ptrInterface, true
	}
	elemInterface := v.Ptr.Elem().Interface()
	if filter(elemInterface) {
		return elemInterface, true
	}
	return nil, false
}

// SliceElements returns a slice of Vals representing the index, value pairs
// from this slice.
// It panics if v is not a slice.
func (v Val) SliceElements(elemNode Node) []Val {
	s := v.Ptr.Elem()
	vals := make([]Val, s.Len())
	for i := 0; i < s.Len(); i++ {
		vals[i] = elemNode.NewKeyedValFrom(reflect.ValueOf(i), s.Index(i))
	}
	return vals
}
