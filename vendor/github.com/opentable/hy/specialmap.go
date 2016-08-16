package hy

import (
	"reflect"

	"github.com/pkg/errors"
)

// SpecialMapNode represents a struct node with GetAll and SetAll methods.
type SpecialMapNode struct {
	NodeBase
	Map    *MapNode
	GetAll func(Val) (reflect.Value, error)
	SetAll func(on Val, to reflect.Value) error
}

// Detect returns nil if base is a struct with appropriate GetAll and SetAll
// methods.
func (SpecialMapNode) Detect(base NodeBase) error {
	if base.Kind != reflect.Struct {
		return errors.Errorf("got kind %s; want struct", base.Kind)
	}
	_, _, err := getAllMapMethods(base.Type)
	return err
}

func getAllMapMethods(baseType reflect.Type) (get, set reflect.Type, err error) {
	get, set, err = getMapMethods(baseType)
	if err == nil {
		return
	}
	if baseType.Kind() == reflect.Ptr {
		baseType = baseType.Elem()
	} else {
		baseType = reflect.PtrTo(baseType)
	}
	return getMapMethods(baseType)
}

func getMapMethods(baseType reflect.Type) (get, set reflect.Type, err error) {
	getAll, ok := baseType.MethodByName("GetAll")
	if !ok {
		return nil, nil, errors.Errorf("no method named GetAll")
	}
	setAll, ok := baseType.MethodByName("SetAll")
	if !ok {
		return nil, nil, errors.Errorf("no method named SetAll")
	}
	return getAll.Type, setAll.Type, nil
}

func getMapType(get, set reflect.Type) (reflect.Type, error) {
	if get.NumIn() != 1 || get.NumOut() != 1 {
		return nil, errors.Errorf("GetAll has wrong signature, want 0 params, 1 return")
	}
	if set.NumIn() != 2 || set.NumOut() != 0 {
		return nil, errors.Errorf("SetAll has wrong signatuer, want 1 param, 0 return")
	}
	mapType := get.Out(0)
	inMapType := set.In(1)
	if mapType != inMapType {
		return nil, errors.Errorf("GetAll and SetAll have different types: %s and %s",
			mapType, inMapType)
	}
	if mapType.Kind() != reflect.Map {
		return nil, errors.Errorf("GetAll/SetAll type (%s) is not a map", mapType)
	}
	return mapType, nil
}

// New returns a new SpecialMapNode.
func (SpecialMapNode) New(base NodeBase, c *Codec) (Node, error) {
	get, set, err := getAllMapMethods(base.Type)
	if err != nil {
		return nil, err
	}
	mapType, err := getMapType(get, set)
	if err != nil {
		return nil, err
	}
	getAllFunc := func(from Val) (reflect.Value, error) {
		out := from.Method("GetAll").Call(nil)
		return out[0], nil
	}
	setAllFunc := func(on Val, to reflect.Value) error {
		on.Method("SetAll").Call([]reflect.Value{to})
		return nil
	}
	innerID, err := NewNodeID(base.Type, mapType, "")
	if err != nil {
		return nil, err
	}
	var n = &SpecialMapNode{
		NodeBase: base,
		GetAll:   getAllFunc,
		SetAll:   setAllFunc,
	}
	var node Node = n
	innerBase := NewNodeBase(innerID, n, nil, &node)
	mapNode, err := (&MapNode{}).New(innerBase, c)
	n.Map = mapNode.(*MapNode)
	return n, err
}

// ChildPathName delegates to MapNode.
func (n *SpecialMapNode) ChildPathName(child Node, key, val reflect.Value) string {
	return n.Map.ChildPathName(child, key, val)
}

// ReadTargets delegates to MapNode.
func (n *SpecialMapNode) ReadTargets(c ReadContext, val Val) error {
	mapVal := n.Map.NewVal()
	if err := n.Map.ReadTargets(c, mapVal); err != nil {
		return err
	}
	return errors.Wrapf(n.SetAll(val, mapVal.Final()), "setting map")
}

// WriteTargets delegates to MapNode.
func (n *SpecialMapNode) WriteTargets(c WriteContext, val Val) error {
	m, err := n.GetAll(val)
	if err != nil {
		return errors.Wrapf(err, "getting map values")
	}
	mapVal := n.Map.NewValFrom(m)
	if err := n.Map.WriteTargets(c, mapVal); err != nil {
		return errors.Wrapf(err, "writing map values")
	}
	return nil
}
