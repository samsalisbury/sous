package hy

import (
	"reflect"

	"github.com/pkg/errors"
)

// StructNode represents a struct to be stored in a file.
type StructNode struct {
	FileNode
	// Fields is a map of simple struct field names to their types.
	Fields map[string]reflect.Type
	// Children is a map of field named to node pointers.
	Children map[string]*Node
}

// Detect returns nil if this base is a struct.
func (StructNode) Detect(base NodeBase) error {
	if base.Kind == reflect.Struct {
		return nil
	}
	return errors.Errorf("got kind %s; want struct", base.Kind)
}

// New creates a new StructNode.
func (StructNode) New(base NodeBase, c *Codec) (Node, error) {
	// Children need a pointer to this node, so create it first.
	n := &StructNode{
		FileNode: FileNode{
			NodeBase: base,
		},
		Fields:   map[string]reflect.Type{},
		Children: map[string]*Node{},
	}
	for i := 0; i < n.Type.NumField(); i++ {
		field, err := NewFieldInfo(n.Type.Field(i)) //tag, Name: field.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "reading field %s.%s", n.Type, n.Type.Field(i).Name)
		}
		if field.Tag.None {
			n.Fields[field.Name] = field.Type
			continue
		}
		if field.Tag.Ignore {
			continue
		}
		childNodeID, err := NewNodeID(n.Type, field.Type, field.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "getting ID for %T.%s", n.Type, field.Name)
		}
		child, err := c.NewNode(n, childNodeID, field)
		if err != nil {
			return nil, errors.Wrapf(err, "analysing %T.%s", n.Type, field.Name)
		}
		if child != nil {
			n.Children[field.Name] = child
		}
	}
	return n, nil
}

// ChildPathName returns the path segment for this node's children.
func (n *StructNode) ChildPathName(child Node, key, val reflect.Value) string {
	name, _ := child.FixedPathName()
	return name
}

// ReadTargets reads targets into struct fields.
func (n *StructNode) ReadTargets(c ReadContext, val Val) error {
	if err := c.Read(val.Ptr.Interface()); err != nil {
		return errors.Wrapf(err, "reading struct fields")
	}
	for fieldName, childPtr := range n.Children {
		childNode := *childPtr
		childPathName := childNode.PathName(val)
		childContext := c.Push(childPathName)
		if !childContext.Exists() {
			continue
		}
		childVal := childNode.NewVal()
		err := childNode.Read(childContext, childVal)
		if err != nil {
			return errors.Wrapf(err, "reading child %s", fieldName)
		}
		val.SetField(fieldName, childVal)
	}
	return nil
}

// WriteTargets generates file targets.
func (n *StructNode) WriteTargets(c WriteContext, val Val) error {
	if !val.ShouldWrite() {
		return nil
	}
	fieldData, any := n.prepareFileData(val)
	if any || n.HasKey {
		if err := c.SetRawValue(fieldData); err != nil {
			return errors.Wrap(err, "writing self")
		}
	}
	if val.IsZero() {
		val = n.NewVal()
	}
	for name, childPtr := range n.Children {
		childNode := *childPtr
		childKey := reflect.ValueOf(name)
		childVal := childNode.NewKeyedValFrom(childKey, val.GetField(name))
		if childVal.IsZero() {
			continue
		}
		childContext := c.Push(childNode.PathName(childVal))
		if err := childNode.Write(childContext, childVal); err != nil {
			return errors.Wrapf(err, "failed to write child %s", name)
		}
	}
	return nil
}

func (n *StructNode) prepareFileData(val Val) (interface{}, bool) {
	if val.IsZero() {
		return nil, false
	}
	// Optimisation which also results in preserving field order.
	if len(n.Children) == 0 {
		return val.Final().Interface(), true
	}
	// Otherwise, we construct a map.
	out := make(map[string]interface{}, len(n.Fields))
	for name := range n.Fields {
		f := val.GetField(name)
		// This excludes unexported fields.
		if !f.CanInterface() {
			continue
		}
		out[name] = f.Interface()
	}
	return out, len(out) != 0
}
