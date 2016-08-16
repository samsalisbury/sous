package hy

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// NodeID identifies a node in the tree.
type NodeID struct {
	// ParentType is the type of this node's parent.
	ParentType,
	// Type is the type of this node.
	Type reflect.Type
	// IsPtr indicates if OwnType is a pointer really.
	IsPtr bool
	// FieldName is the name of the parent field containing this node. FieldName
	// will be empty unless ParentType is a struct.
	FieldName string
}

func normalise(original reflect.Type) (normal reflect.Type, k reflect.Kind, ptr bool, err error) {
	normal = original
	k = original.Kind()
	if k == reflect.Ptr {
		ptr = true
		normal = normal.Elem()
		k = normal.Kind()
		if k == reflect.Ptr {
			err = errors.New("cannot analyse pointer to pointer")
		}
	}
	if k == reflect.Interface {
		err = errors.New("cannot analyse kind interface")
	}
	return
}

// NewNodeID creates a new node ID.
func NewNodeID(parentType, typ reflect.Type, fieldName string) (NodeID, error) {
	t, _, isPtr, err := normalise(typ)
	if err != nil {
		return NodeID{}, err
	}
	return NodeID{
		ParentType: parentType,
		Type:       t,
		IsPtr:      isPtr,
		FieldName:  fieldName,
	}, nil
}

func (id NodeID) String() string {
	ptr := ""
	if id.IsPtr {
		ptr = "*"
	}
	parent := "nil"
	if id.ParentType != nil {
		parent = id.ParentType.String()
	}
	return fmt.Sprintf("%s%s.%s(%s)", ptr, parent, id.FieldName, id.Type)
}
