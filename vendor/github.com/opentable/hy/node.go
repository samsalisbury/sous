package hy

import "reflect"

// Node represents a generic node in the structure.
type Node interface {
	// Detect returns nil if this node can handle this base type.
	Detect(NodeBase) error
	// New returns a new instance of a node.
	New(NodeBase, *Codec) (Node, error)
	// ID returns this node's ID.
	ID() NodeID
	// FixedPathName returns the indubitable path segment name of this node.
	FixedPathName() (string, bool)
	// ChildPathName returns the path segment for children of this node.
	// If the node's parent is a map or slice, both key and val will have
	// valid values, with val having the same type as this node.
	// If the node's parent is a map, then the key will be a value of the
	// parent's key type.
	// If the node's parent is a slice, then key will be an int value
	// representing the index of this element.
	// If the node's parent is a struct, then key will be an invalid value,
	// and val will be the value of that struct field.
	ChildPathName(child Node, key, val reflect.Value) string
	// PathName returns the path name of this node. Implemented in NodeBase.
	PathName(Val) string
	// WriteTargets writes file targets for this node to the context.
	WriteTargets(c WriteContext, val Val) error
	// Write writes file targets for this node to the context by first ensuring
	// val is not a pointer and then calling WriteTargets.
	Write(c WriteContext, val Val) error
	// Read wraps ReadTargets and takes care of pointers.
	Read(c ReadContext, val Val) error
	// ReadTargets reads key from contexts and returns its value.
	ReadTargets(c ReadContext, val Val) error

	NewVal() Val
	NewValFrom(reflect.Value) Val
	NewKeyedVal(key reflect.Value) Val
	NewKeyedValFrom(key, val reflect.Value) Val
}
