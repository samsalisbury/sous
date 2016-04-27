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
	"path/filepath"
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
	}
	target struct {
		path string
		val  reflect.Value
		typ  reflect.Type
		// name is the name of this value in its parent struct or map
		name string
		// subTargets includes both map and slice element targets, as well as
		// struct field targets.
		subTargets targets
	}
	targets []*target
)

func NewUnmarshaler(unmarshalFunc func([]byte, interface{}) error) Unmarshaler {
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
	return ctx{path, u.UnmarshalFunc}.unmarshalDir(v)
}

func (ts targets) unmarshalAll(parent *reflect.Value, unmarshalFunc func([]byte, interface{}) error) error {
	for _, t := range ts {
		log.Printf("Target: %s\n", t.path)
		iface := t.val.Interface()
		if isFile(t.path) {
			if t.val.Kind() != reflect.Ptr || t.val.Elem().Kind() != reflect.Struct {
				return fmt.Errorf("tried to unmarshal file %s to %T; want a pointer to struct", t.path, iface)
			}
			log.Printf("Path: %s; Type: %s; ValType: %s; IfaceType: %s\n", t.path, t.typ, t.val.Type(), reflect.TypeOf(t.val.Interface()))
			b, err := ioutil.ReadFile(t.path)
			if err != nil {
				return err
			}
			if err := unmarshalFunc(b, iface); err != nil {
				return err
			}
			log.Printf("Unmarshalled: val: %v; type: %T", iface, iface)
		}
		if parent != nil {
			parentTypeErr := fmt.Errorf("parent was %T; want a pointer or map", parent.Interface())
			switch parent.Kind() {
			default:
				return parentTypeErr
			case reflect.Ptr:
				if parent.Elem().Kind() != reflect.Struct {
					return parentTypeErr
				}
				log.Printf("Setting field %s on %s\n", t.name, parent.Elem().Type())
				f := parent.Elem().FieldByName(t.name)
				f.Set(*getConcreteValRef(t.val))
			case reflect.Map:
				if parent.Type().Key().Kind() != reflect.String {
					return parentTypeErr
				}
				log.Printf("Setting key %q on %s\n", t.name, parent.Type())
				if parent.IsNil() {
					pvp := reflect.MakeMap(parent.Type())
					log.Printf("Parent was nil, setting empty map of %s\n", parent.Type())
					parent.Set(pvp)
				}
				parent.SetMapIndex(reflect.ValueOf(t.name), t.val.Elem())
			}
		}
		if len(t.subTargets) != 0 {
			if err := t.subTargets.unmarshalAll(&t.val, unmarshalFunc); err != nil {
				return err
			}
		}
	}
	return nil
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

func (c ctx) unmarshalDir(v interface{}) error {
	targets, err := c.getStructTargets(v)
	if err != nil {
		return err
	}
	return targets.unmarshalAll(nil, c.unmarshal)
}

func isFile(path string) bool {
	return strings.HasSuffix(path, ".yaml")
}

func (c ctx) getStructTargets(v interface{}) (targets, error) {
	if v == nil {
		panic("hy tried to unmarshal to nil, please report this")
	}
	val := reflect.ValueOf(v)
	k := val.Kind()
	if k != reflect.Ptr {
		return nil, fmt.Errorf("getStructTargets passed non-pointer")
	}
	typ := val.Type().Elem()
	nf := typ.NumField()
	t := &target{path: c.path, val: val}
	t.subTargets = targets{}
	for i := 0; i < nf; i++ {
		f := typ.Field(i)
		tag := f.Tag.Get("hy")
		if tag != "" {
			ts, err := c.getTarget(f.Name, tag, f.Type, val.Elem().Field(i))
			if err != nil {
				return nil, err
			}
			t.subTargets = append(t.subTargets, ts...)
		}
	}
	return targets{t}, nil
}

func (c ctx) getDirTargets(source, name string, typ reflect.Type, val reflect.Value) (targets, error) {
	if typ.Kind() != reflect.Map {
		return nil, fmt.Errorf("directory targets only accept maps for now")
	}
	elemType, err := getElemType(typ)
	if err != nil {
		return nil, err
	}
	c = c.enter(source)
	yamlFiles, err := filepath.Glob(c.enter("*.yaml").path)
	if err != nil {
		return nil, err
	}
	subTargets := make(targets, len(yamlFiles))
	for i, filename := range yamlFiles {
		filename = strings.TrimPrefix(filename, c.path)
		name := strings.TrimPrefix(strings.TrimSuffix(filename, ".yaml"), "/")
		subTargets[i], err = c.getFileTarget(filename, name, elemType, newValue(elemType))
		if err != nil {
			return nil, err
		}
	}
	t := &target{path: c.path, name: name, typ: typ, val: val, subTargets: subTargets}
	return targets{t}, nil
}

func (c ctx) getTarget(name, tag string, typ reflect.Type, val reflect.Value) (targets, error) {
	source := strings.Split(tag, ",")[0]
	if strings.HasSuffix(source, ".yaml") {
		t, err := c.getFileTarget(source, name, typ, val)
		return targets{t}, err
	}
	if strings.HasSuffix(source, "/") {
		return c.getDirTargets(source, name, typ, val)
	}
	if strings.HasSuffix(source, "/**") {
		return c.getTreeTargets(source, name, typ, val)
	}
	return nil, fmt.Errorf("%s.%s has hy tag %q; source does not end with .yaml, /, nor /**", typ, name, tag)
}

func (c ctx) getTreeTargets(source, name string, typ reflect.Type, val reflect.Value) (targets, error) {
	panic("this should not be getting called yet")
	elemType, err := getElemType(typ)
	if err != nil {
		return nil, err
	}
	source = strings.TrimSuffix(source, "**")
	targets := targets{}
	c = c.enter(source)
	err = filepath.Walk(c.path, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		t, err := c.getFileTarget(path, path, elemType, newValue(elemType))
		if err != nil {
			return err
		}
		targets = append(targets, t)
		return nil
	})
	return targets, nil
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

func (c ctx) getFileTarget(source, name string, typ reflect.Type, val reflect.Value) (*target, error) {
	c = c.enter(source)
	v := reflect.New(typ)
	v.Elem().Set(val)
	return &target{path: c.path, name: name, val: v, typ: typ}, nil
}

func (c ctx) enter(path string) ctx {
	return ctx{path: filepath.Join(c.path, path)}
}
