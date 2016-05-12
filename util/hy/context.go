package hy

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type (
	ctx struct {
		path      string
		unmarshal func([]byte, interface{}) error
		marshal   func(interface{}) ([]byte, error)
	}
	walkFunc func(name, tag string, val reflect.Value) (*target, error)
)

func (c ctx) writeStructTargets(v interface{}) (targets, error) {
	return c.walkStruct(v, c.writeTarget)
}

func (c ctx) walkStruct(v interface{}, walkFunc walkFunc) (targets, error) {
	if v == nil {
		panic("hy tried to unmarshal to nil, please report this")
	}
	val := reflect.ValueOf(v)
	k := val.Kind()
	if k != reflect.Ptr {
		panic("getStructTargets passed non-pointer")
	}
	typ := val.Type().Elem()
	nf := typ.NumField()
	subTargets := targets{}
	for i := 0; i < nf; i++ {
		f := typ.Field(i)
		tag := f.Tag.Get("hy")
		if tag != "" {
			t, err := walkFunc(f.Name, tag, val.Elem().Field(i))
			if err != nil {
				return nil, err
			}
			subTargets = append(subTargets, t)
		}
	}
	t := c.makeTarget("", val, subTargets)
	return targets{t}, nil
}

func (c ctx) getStructTargets(v interface{}) (targets, error) {
	return c.walkStruct(v, c.readTarget)
}

func (c ctx) readDirTarget(source, name string, val reflect.Value) (*target, error) {
	typ := val.Type()
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
		subTargets[i], err = c.getFileTarget(filename, pathToName(filename), newValue(elemType))
		if err != nil {
			return nil, err
		}
	}
	return c.makeTarget(name, val, subTargets), nil
}

func (c ctx) readTree(elemType reflect.Type) (targets, error) {
	ts := targets{}
	err := filepath.Walk(c.path, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		path = strings.TrimPrefix(path, c.path)
		t, err := c.getFileTarget(path, pathToName(path), newValue(elemType))
		if err != nil {
			return err
		}
		ts = append(ts, t)
		return nil
	})
	return ts, err
}

func (c ctx) readTreeTarget(source, name string, val reflect.Value) (*target, error) {
	typ := val.Type()
	elemType, err := getElemType(typ)
	if err != nil {
		return nil, err
	}
	source = strings.TrimSuffix(source, "**")
	c = c.enter(source)
	subTargets, err := c.readTree(elemType)
	return c.makeTarget(name, val, subTargets), nil
}

func (c ctx) writeDirTarget(source, name string, val reflect.Value) (*target, error) {
	return c.writeTreeTarget(source, name, val)
}

func (c ctx) writeTreeTarget(source, name string, val reflect.Value) (*target, error) {
	source = strings.TrimSuffix(source, "**")
	c = c.enter(source)
	m := reflect.MakeMap(reflect.TypeOf(map[string]interface{}{}))
	for _, k := range val.MapKeys() {
		elemVal := val.MapIndex(k)
		m.SetMapIndex(k, elemVal)
	}
	subTargets, err := c.writeTree(m.Interface().(map[string]interface{}))
	if err != nil {
		return nil, err
	}
	return c.makeTarget(name, val, subTargets), nil
}

func (c ctx) writeTree(m map[string]interface{}) (targets, error) {
	ts := make(targets, len(m))
	i := 0
	for name, val := range m {
		ts[i] = c.enter(name).makeTarget(name, reflect.ValueOf(val), nil)
		i++
	}
	return ts, nil
}

func (c ctx) makeTarget(name string, val reflect.Value, subTargets targets) *target {
	return &target{
		path:          c.path,
		name:          name,
		typ:           val.Type(),
		val:           val,
		subTargets:    subTargets,
		unmarshalFunc: c.unmarshal,
		marshalFunc:   c.marshal,
	}
}

func (c ctx) readTarget(name, tag string, val reflect.Value) (*target, error) {
	source := strings.Split(tag, ",")[0]
	if strings.HasSuffix(source, ".yaml") {
		return c.getFileTarget(source, name, val)
	}
	if strings.HasSuffix(source, "/") {
		return c.readDirTarget(source, name, val)
	}
	if strings.HasSuffix(source, "/**") {
		return c.readTreeTarget(source, name, val)
	}
	return nil, fmt.Errorf("%s.%s has hy tag %q; source does not end with .yaml, /, nor /**", val.Type(), name, tag)
}

func (c ctx) writeTarget(name, tag string, val reflect.Value) (*target, error) {
	source := strings.Split(tag, ",")[0]
	if strings.HasSuffix(source, ".yaml") {
		return c.getFileTarget(source, name, val)
	}
	if strings.HasSuffix(source, "/") {
		return c.writeDirTarget(source, name, val)
	}
	if strings.HasSuffix(source, "/**") {
		return c.writeTreeTarget(source, name, val)
	}
	return nil, fmt.Errorf("%s.%s has hy tag %q; source does not end with .yaml, /, nor /**", val.Type(), name, tag)
}

func (c ctx) getFileTarget(source, name string, val reflect.Value) (*target, error) {
	c = c.enter(source)
	v := reflect.New(val.Type())
	v.Elem().Set(val)
	return c.makeTarget(name, v, nil), nil
}

func (c ctx) enter(path string) ctx {
	return ctx{
		path:      filepath.Join(c.path, path),
		unmarshal: c.unmarshal,
		marshal:   c.marshal,
	}
}

func pathToName(path string) string {
	return strings.TrimPrefix(strings.TrimSuffix(path, ".yaml"), "/")
}
