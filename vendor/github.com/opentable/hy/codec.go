package hy

import (
	"reflect"

	"github.com/pkg/errors"
)

// Codec provides the primary encoding and decoding facility of this package.
// Codecs should be re-used to take advantage of their caching of reflection
// data.
type Codec struct {
	// MarshalFunc is used to write file data. The signature matches
	// json.Marshal as well as many other standard marshalers.
	MarshalFunc func(interface{}) ([]byte, error)
	// UnmarshalFunc is used to read file data. The signature matches
	// json.Unmarshal as well as many other standard unmarshalers.
	UnmarshalFunc func([]byte, interface{}) error
	// FileExtension is the extension to use for reading and writing files. It
	// must not be empty.
	FileExtension,
	// RootFileName is the file name (without extension) to use for any fields
	// left over from the root struct, after its hy-tagged fields have been
	// taken out.
	// It defaults to _ if not set.
	RootFileName string

	nodes      NodeSet
	writer     FileWriter
	reader     FileReader
	treeReader *FileTreeReader
}

// NewCodec creates a new codec.
func NewCodec(configure ...func(*Codec)) *Codec {
	c := &Codec{nodes: NewNodeSet()}
	for _, cfg := range configure {
		if cfg == nil {
			continue
		}
		cfg(c)
	}
	if c.UnmarshalFunc == nil && c.MarshalFunc == nil && c.FileExtension == "" {
		c.FileExtension = "json"
	}
	if c.UnmarshalFunc == nil {
		c.UnmarshalFunc = JSONWriter.UnmarshalFunc
	}
	if c.MarshalFunc == nil {
		c.MarshalFunc = JSONWriter.MarshalFunc
	}
	if c.RootFileName == "" {
		c.RootFileName = "_"
	}

	marshaler := FileMarshaler{
		UnmarshalFunc: c.UnmarshalFunc,
		MarshalFunc:   c.MarshalFunc,
		FileExtension: c.FileExtension,
		RootFileName:  c.RootFileName,
	}
	if c.writer == nil {
		c.writer = marshaler
	}
	if c.reader == nil {
		c.reader = marshaler
	}
	if c.treeReader == nil {
		c.treeReader = NewFileTreeReader(c.FileExtension, c.RootFileName)
	}
	return c
}

// NodeTypes contains the set of nodes types in order of preference.
// Earlier types will be detected before later ones.
var NodeTypes = []Node{
	&SpecialMapNode{}, &StructNode{}, &FileNode{}, &MapNode{}, &SliceNode{},
}

// NewNode creates a new node.
func (c *Codec) NewNode(parent Node, id NodeID, field *FieldInfo) (*Node, error) {
	n, new := c.nodes.Register(id)
	if !new {
		return n, nil
	}
	var err error
	base := NewNodeBase(id, parent, field, n)
	for _, nt := range NodeTypes {
		if err := nt.Detect(base); err == nil {
			*n, err = nt.New(base, c)
			if err != nil {
				continue
			}
			return n, err
		}
	}
	return n, errors.Wrapf(err, "analysing %s failed; no nodes matched", id)
}

func (c *Codec) Read(prefix string, root interface{}) error {
	rootNode, err := c.Analyse(root)
	if err != nil {
		return errors.Wrapf(err, "analysing structure")
	}
	targets, err := c.treeReader.ReadTree(prefix)
	if err != nil {
		return errors.Wrapf(err, "reading tree at %q", prefix)
	}
	rc := NewReadContext(prefix, targets, c.reader)
	rootVal := rootNode.NewValFrom(reflect.ValueOf(root))
	if err := rootNode.Read(rc, rootVal); err != nil {
		return errors.Wrapf(err, "reading root")
	}
	reflect.ValueOf(root).Elem().Set(rootVal.Ptr.Elem())
	return nil
}

func (c *Codec) Write(prefix string, root interface{}) error {
	rootNode, err := c.Analyse(root)
	if err != nil {
		return errors.Wrapf(err, "analysing structure")
	}
	wc := NewWriteContext()
	v := reflect.ValueOf(root)
	val := rootNode.NewValFrom(v)
	if err := rootNode.Write(wc, val); err != nil {
		return errors.Wrapf(err, "generating write targets")
	}
	for _, t := range wc.targets.Snapshot() {
		if err := c.writer.WriteFile(prefix, t); err != nil {
			return errors.Wrapf(err, "writing target %q", t.Path())
		}
	}
	return nil
}

// Analyse analyses a tree starting at root.
func (c *Codec) Analyse(root interface{}) (Node, error) {
	if root == nil {
		return nil, errors.New("cannot analyse nil")
	}
	t := reflect.TypeOf(root)
	id, err := NewNodeID(nil, t, "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to analyse %T", root)
	}
	t, k, _, err := normalise(t)
	if err != nil {
		return nil, err
	}
	isLeaf := (k != reflect.Struct && k != reflect.Map && k != reflect.Slice)
	if isLeaf {
		return nil, errors.Errorf("failed to analyse %s: cannot analyse kind %s",
			id.Type, id.Type.Kind())
	}
	n, err := c.NewNode(nil, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to analyse %T", root)
	}
	return *n, err
}
