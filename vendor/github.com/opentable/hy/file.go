package hy

import (
	"reflect"

	"github.com/pkg/errors"
)

// A FileNode represents a node to be stored in a file.
type FileNode struct {
	NodeBase
}

// Detect always returns nil.
func (FileNode) Detect(base NodeBase) error {
	if base.Field.Tag.IsDir {
		return errors.Errorf("got directory, want file")
	}
	return nil
}

// New returns a new file node and nil error.
func (FileNode) New(base NodeBase, _ *Codec) (Node, error) {
	return &FileNode{NodeBase: base}, nil
}

// ChildPathName returns an empty string (file targets don't have children).
func (n *FileNode) ChildPathName(child Node, key, val reflect.Value) string {
	return ""
}

// ReadTargets reads a single file target.
func (n *FileNode) ReadTargets(c ReadContext, val Val) error {
	err := c.Read(val.Ptr.Interface())
	return errors.Wrapf(err, "reading file")
}

// WriteTargets returns the write target for this file.
func (n *FileNode) WriteTargets(c WriteContext, val Val) error {
	return errors.Wrap(c.SetValue(val), "writing file target")
}
