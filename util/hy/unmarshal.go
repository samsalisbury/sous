// hy is a two-way hierarchical YAML parser.
//
// hy allows you to read and write YAML files in a directory hierarchy
// to and from go structs. It uses tags to define the locations of
// YAML files and directories containing YAML files, and some simple
// mapping of filenames to string values, used during packing and unpacking.
package hy

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
)

type (
	Unmarshaler struct {
		UnmarshalFunc func([]byte, interface{}) error
	}
	ctx struct {
		path      string
		unmarshal func([]byte, interface{}) error
		marshal   func(interface{}) ([]byte, error)
	}
	target struct {
		path string
		val  reflect.Value
		typ  reflect.Type
		// name is the name of this value in its parent struct or map
		name string
		// subTargets includes both map and slice element targets, as well as
		// struct field targets.
		subTargets    targets
		unmarshalFunc func([]byte, interface{}) error
	}
	targets []*target
)

func NewUnmarshaler(unmarshalFunc func([]byte, interface{}) error) Unmarshaler {
	if unmarshalFunc == nil {
		panic("unmarshalFunc must not be nil")
	}
	return Unmarshaler{unmarshalFunc}
}

func (u Unmarshaler) Unmarshal(path string, v interface{}) error {
	if v == nil {
		return fmt.Errorf("hy cannot unmarshal to nil")
	}
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !s.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return ctx{path, u.UnmarshalFunc, nil}.unmarshalDir(v)
}

func (c ctx) unmarshalDir(v interface{}) error {
	targets, err := c.getStructTargets(v)
	if err != nil {
		return err
	}
	return targets.unmarshalAll(nil)
}

func (ts targets) unmarshalAll(parent *reflect.Value) error {
	for _, t := range ts {
		if err := t.unmarshal(parent); err != nil {
			return err
		}
	}
	return nil
}

func (t target) unmarshal(parent *reflect.Value) error {
	log.Printf("Target: %s\n", t.path)
	iface := t.val.Interface()
	if isFile(t.path) {
		if err := t.unmarshalFile(iface); err != nil {
			return err
		}
	}
	if parent != nil {
		if err := t.insertIntoParent(parent); err != nil {
			return err
		}
	}
	if len(t.subTargets) != 0 {
		if err := t.subTargets.unmarshalAll(&t.val); err != nil {
			return err
		}
	}
	return nil
}

func parentTypeError(parent *reflect.Value) error {
	return fmt.Errorf("parent was %s; want pointer or map[string]T", parent.Type())
}

func (t target) insertIntoParent(parent *reflect.Value) error {
	switch parent.Kind() {
	default:
		return parentTypeError(parent)
	case reflect.Ptr:
		if parent.Elem().Kind() != reflect.Struct {
			return parentTypeError(parent)
		}
		log.Printf("Setting field %s on %s\n", t.name, parent.Elem().Type())
		f := parent.Elem().FieldByName(t.name)
		f.Set(*getConcreteValRef(t.val))
	case reflect.Map:
		if parent.Type().Key().Kind() != reflect.String {
			return parentTypeError(parent)
		}
		log.Printf("Setting key %q on %s\n", t.name, parent.Type())
		if parent.IsNil() {
			pvp := reflect.MakeMap(parent.Type())
			log.Printf("Parent was nil, setting empty map of %s\n", parent.Type())
			parent.Set(pvp)
		}
		parent.SetMapIndex(reflect.ValueOf(t.name), t.val.Elem())
	}
	return nil
}

func (t target) unmarshalFile(iface interface{}) error {
	if t.val.Kind() != reflect.Ptr || t.val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("tried to unmarshal file %s to %T; want a pointer to struct", t.path, iface)
	}
	log.Printf("Path: %s; Type: %s; ValType: %s; IfaceType: %s\n", t.path, t.typ, t.val.Type(), reflect.TypeOf(t.val.Interface()))
	b, err := ioutil.ReadFile(t.path)
	if err != nil {
		return err
	}
	if err := t.unmarshalFunc(b, iface); err != nil {
		return err
	}
	log.Printf("Unmarshalled: val: %v; type: %T", iface, iface)
	return nil
}

// getElemType tries to get element type of a map or slice, and if that type is
// a pointer, gets the element type of the pointer instead. If the elem type is
// pointer to pointer, or the type passed in is not a map or slice,  returns an
// error.
func getElemType(typ reflect.Type) (reflect.Type, error) {
	k := typ.Kind()
	switch k {
	default:
		return nil, fmt.Errorf("directory target not allowed for type %s; want map or slice", typ)
	case reflect.Slice, reflect.Map:
		break
	}
	elemType := typ.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() == reflect.Ptr {
		return nil, fmt.Errorf("%s containing %s not supported", k, typ.Elem())
	}
	return elemType, nil
}

func newValue(typ reflect.Type) reflect.Value {
	if typ.Kind() == reflect.Ptr {
		panic("newValue passed a pointer type")
	}
	return reflect.New(typ).Elem()
}

func getConcreteValRef(v reflect.Value) *reflect.Value {
	switch v.Kind() {
	default:
		return &v
	case reflect.Ptr:
		e := v.Elem()
		return &e
	}
}

func pathToName(path string) string {
	return strings.TrimPrefix(strings.TrimSuffix(path, ".yaml"), "/")
}

func isFile(path string) bool {
	return strings.HasSuffix(path, ".yaml")
}
