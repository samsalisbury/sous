package hy

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
)

// A SliceNode represents a slice to be stored in a directory.
type SliceNode struct {
	*DirNodeBase
}

// Detect returns nil if this base is a slice.
func (SliceNode) Detect(base NodeBase) error {
	if base.Kind == reflect.Slice {
		return nil
	}
	return errors.Errorf("got kind %s; want slice", base.Kind)
}

// New returns a new slice node.
func (SliceNode) New(base NodeBase, c *Codec) (Node, error) {
	n := &SliceNode{&DirNodeBase{NodeBase: base}}
	return n, errors.Wrap(n.AnalyseElemNode(n, c), "analysing slice element node")
}

// ChildPathName returns the slice index as a string.
func (n *SliceNode) ChildPathName(child Node, key, val reflect.Value) string {
	return fmt.Sprint(key)
}

// ReadTargets reads targets into slice indicies.
func (n *SliceNode) ReadTargets(c ReadContext, val Val) error {
	list := c.List()
	for _, indexStr := range list {
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return errors.Wrapf(err, "converting %q to int", indexStr)
		}
		elemKey := reflect.ValueOf(index)
		elem := *n.ElemNode
		elemContext := c.Push(indexStr)
		elemVal := elem.NewKeyedVal(elemKey)
		if err := elem.Read(elemContext, elemVal); err != nil {
			return errors.Wrapf(err, "reading index %d", index)
		}
		val.Append(elemVal)
	}
	return nil
}

// WriteTargets writes all the elements of the slice.
func (n *SliceNode) WriteTargets(c WriteContext, val Val) error {
	elemNode := *n.ElemNode
	for i, childVal := range val.SliceElements(elemNode) {
		childContext := c.Push(elemNode.PathName(childVal))
		if err := elemNode.Write(childContext, childVal); err != nil {
			return errors.Wrapf(err, "writing slice index %d failed", i)
		}
	}
	return nil
}
