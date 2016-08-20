package hy

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// A MapNode represents a map node to be stored in a directory.
type MapNode struct {
	*DirNodeBase
	KeyType reflect.Type
	// MarshalKey gets a string from the key.
	MarshalKey func(key Val) string
	// UnmarshalKey sets a key from a string.
	UnmarshalKey func(key string, val reflect.Value) error
}

// Detect returns nil if this base is a map.
func (MapNode) Detect(base NodeBase) error {
	if base.Kind == reflect.Map {
		return nil
	}
	return errors.Errorf("got kind %s; want map", base.Kind)
}

// New returns a new MapNode.
func (MapNode) New(base NodeBase, c *Codec) (Node, error) {
	n := &MapNode{
		DirNodeBase: &DirNodeBase{
			NodeBase: base,
		},
		KeyType: base.Type.Key(),
	}
	switch n.KeyType.Kind() {
	default:
		// Note: this can be made much more efficient by implementing separate
		// funcs per pointer/non-pointer version of marshal and unmarshal.
		n.MarshalKey = defaultMarshalKey
		n.UnmarshalKey = defaultUnmarshalKey
	case reflect.String:
		n.MarshalKey = func(key Val) string {
			return fmt.Sprint(key.Final())
		}
		n.UnmarshalKey = func(s string, key reflect.Value) error {
			key.Set(reflect.ValueOf(s))
			return nil
		}
	}
	return n, errors.Wrap(n.AnalyseElemNode(n, c), "analysing map element node")
}

func defaultMarshalKey(key Val) string {
	i, ok := key.Interface(func(v interface{}) bool {
		_, ok := v.(encoding.TextMarshaler)
		return ok
	})
	if !ok {
		panic(errors.Errorf("%s does not implement %s", key.Ptr.Elem().Type(), tmType))
	}
	tm := i.(encoding.TextMarshaler)
	b, err := tm.MarshalText()
	if err != nil {
		panic(errors.Errorf("marshal failed: %s", err.Error()))
	}
	return string(b)
}

func defaultUnmarshalKey(s string, key reflect.Value) error {
	if key.Kind() != reflect.Ptr {
		// Unmarshaling is ineffective on non-pointer receivers, so don't look
		// for it.
		key = key.Addr()
	}
	if key.IsNil() {
		key.Set(reflect.New(key.Type().Elem()))
	}
	i := key.Interface()
	tu, ok := i.(encoding.TextUnmarshaler)
	if !ok {
		return errors.Errorf("%T does not implement %s", i, tuType)
	}
	return errors.Wrapf(tu.UnmarshalText([]byte(s)), "unmarshaling %q", s)
}

var tmType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
var tuType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// ChildPathName returns the key as a string.
func (n *MapNode) ChildPathName(child Node, key, val reflect.Value) string {
	keyVal := NewFreeValFrom(key)
	return n.MarshalKey(keyVal)
}

// ReadTargets reads targets into map entries.
func (n *MapNode) ReadTargets(c ReadContext, val Val) error {
	list := c.List()
	for _, keyStr := range list {
		keyVal := reflect.New(n.KeyType).Elem()
		if err := n.UnmarshalKey(keyStr, keyVal); err != nil {
			return errors.Wrapf(err, "unmarshaling key")
		}
		elem := *n.ElemNode
		elemContext := c.Push(keyStr)
		elemVal := elem.NewKeyedVal(keyVal)
		err := elem.Read(elemContext, elemVal)
		// Set key field.
		if n.Field != nil && n.Field.KeyField != "" {
			n.Field.SetKeyFunc.Call([]reflect.Value{elemVal.Ptr, elemVal.Key})
		}
		if err != nil {
			return errors.Wrapf(err, "reading child %s", keyStr)
		}
		// TODO: Don't calculate these values every time.
		if reflect.DeepEqual(elemVal.Ptr.Elem().Interface(), reflect.New(elemVal.Ptr.Type().Elem()).Elem().Interface()) {
			nv := reflect.New(elemVal.Ptr.Type()).Elem()
			val.Ptr.Elem().SetMapIndex(elemVal.Key, nv)
		} else {
			val.Ptr.Elem().SetMapIndex(elemVal.Key, elemVal.Final())
		}
	}
	return nil
}

// WriteTargets writes all map elements.
func (n *MapNode) WriteTargets(c WriteContext, val Val) error {
	if !val.ShouldWrite() {
		return nil
	}
	elemNode := *n.ElemNode
	for _, elemVal := range val.MapElements(elemNode) {
		// Set key field.
		if n.Field != nil && n.Field.KeyField != "" {
			n.Field.SetKeyFunc.Call([]reflect.Value{elemVal.Ptr, elemVal.Key})
		}
		childContext := c.Push(elemNode.PathName(elemVal))
		if err := elemNode.Write(childContext, elemVal); err != nil {
			return errors.Wrapf(err, "writing map index %q failed",
				fmt.Sprint(elemVal.Key))
		}
	}
	return nil
}
