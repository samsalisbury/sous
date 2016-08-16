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
