package hy

import (
	"reflect"

	"github.com/pkg/errors"
)

// NodeBase is a node in an analysis.
type NodeBase struct {
	NodeID
	// Parent is the parent of this node. It is nil only for the root node.
	Parent Node
	// FieldInfo is the field info for this node.
	Field *FieldInfo
	// Zero is a zero value of this node's Type.
	Zero interface{}
	// HasKey indicates if this type has a key (e.g. maps and slices)
	HasKey bool
	// Kind is the kind of NodeID.Type.
	Kind reflect.Kind
	// self is a pointer to the node based on this node base. This means more
	// common functionality can be handled by NodeBase, by allowing it to call
	// methods on it's differentiated self.
	//
	// self is only safe to use after analysis is complete.
	self *Node
}

// ID returns the ID of this node base.
func (base NodeBase) ID() NodeID {
	return base.NodeID
}

// NewNodeBase returns a new NodeBase.
func NewNodeBase(id NodeID, parent Node, field *FieldInfo, self *Node) NodeBase {
	var parentKind reflect.Kind
	if parent != nil {
		parentKind = parent.ID().Type.Kind()
	}
	var zero interface{}
	if !id.IsPtr {
		zero = reflect.Zero(id.Type).Interface()
	}
	return NodeBase{
		NodeID: id,
		Parent: parent,
		Field:  field,
		Zero:   zero,
		HasKey: parentKind == reflect.Map || parentKind == reflect.Slice,
		Kind:   id.Type.Kind(),
		self:   self,
	}
}

// NewVal creates a new Val of this node's type.
func (base NodeBase) NewVal() Val {
	ptr := reflect.New(base.Type)
	if base.Type.Kind() == reflect.Map {
		ptr.Elem().Set(reflect.MakeMap(base.Type))
	}
	return Val{
		Base:  &base,
		Ptr:   ptr,
		IsPtr: base.IsPtr,
	}
}

// NewKeyedVal is similar to NewVal but adds an associated key.
func (base NodeBase) NewKeyedVal(key reflect.Value) Val {
	val := base.NewVal()
	val.Key = key
	return val
}

// NewValFrom creates a Val from an existing value.
func (base NodeBase) NewValFrom(v reflect.Value) Val {
	val := base.NewVal()
	if v.Kind() == reflect.Ptr {
		val.Ptr = v
		return val
	}
	if v.CanAddr() {
		val.Ptr = v.Addr()
		return val
	}
	val.Ptr = reflect.New(base.Type)
	val.Ptr.Elem().Set(v)
	return val
}

// NewKeyedValFrom is similar to NewValFrom but adds an associated key.
func (base NodeBase) NewKeyedValFrom(k, v reflect.Value) Val {
	val := base.NewValFrom(v)
	val.Key = k
	return val
}

func (base NodeBase) Read(c ReadContext, val Val) error {
	return errors.Wrapf((*base.self).ReadTargets(c, val), "reading node")
}

func (base NodeBase) Write(c WriteContext, val Val) error {
	if !val.ShouldWrite() {
		return nil
	}
	return (*base.self).WriteTargets(c, val)
}

// PathName returns the path name segment of this node by querying its tag,
// field name, and parent's ChildPathName func.
func (base NodeBase) PathName(val Val) string {
	if fixedName, ok := base.FixedPathName(); ok {
		return fixedName
	}
	if base.Parent == nil {
		return ""
	}
	return base.Parent.ChildPathName(*base.self, val.Key, val.Ptr)
}

// FixedPathName returns the fixed path name of this node.
// If there is no fixed path name, returns empty string and false.
// Otherwise returns the fixed path name and true.
func (base NodeBase) FixedPathName() (string, bool) {
	if base.Field == nil {
		return "", false
	}
	if base.Field.PathName != "" {
		return base.Field.PathName, true
	}
	if base.FieldName != "" {
		return base.FieldName, true
	}
	return "", false
}
