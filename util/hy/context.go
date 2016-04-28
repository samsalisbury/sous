package hy

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

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
	subTargets := targets{}
	for i := 0; i < nf; i++ {
		f := typ.Field(i)
		tag := f.Tag.Get("hy")
		if tag != "" {
			ts, err := c.getTarget(f.Name, tag, val.Elem().Field(i))
			if err != nil {
				return nil, err
			}
			subTargets = append(subTargets, ts...)
		}
	}
	t := c.makeTarget("", val, subTargets)
	return targets{t}, nil
}

func (c ctx) getDirTargets(source, name string, val reflect.Value) (targets, error) {
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
	t := c.makeTarget(name, val, subTargets)
	return targets{t}, nil
}

func (c ctx) getTreeTargets(source, name string, val reflect.Value) (targets, error) {
	typ := val.Type()
	elemType, err := getElemType(typ)
	if err != nil {
		return nil, err
	}
	source = strings.TrimSuffix(source, "**")
	subTargets := targets{}
	c = c.enter(source)
	err = filepath.Walk(c.path, func(path string, f os.FileInfo, err error) error {
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
		subTargets = append(subTargets, t)
		return nil
	})
	t := c.makeTarget(name, val, subTargets)
	return targets{t}, nil
}

func (c ctx) makeTarget(name string, val reflect.Value, subTargets targets) *target {
	return &target{
		path:          c.path,
		name:          name,
		typ:           val.Type(),
		val:           val,
		subTargets:    subTargets,
		unmarshalFunc: c.unmarshal,
	}
}

func (c ctx) getTarget(name, tag string, val reflect.Value) (targets, error) {
	source := strings.Split(tag, ",")[0]
	if strings.HasSuffix(source, ".yaml") {
		t, err := c.getFileTarget(source, name, val)
		return targets{t}, err
	}
	if strings.HasSuffix(source, "/") {
		return c.getDirTargets(source, name, val)
	}
	if strings.HasSuffix(source, "/**") {
		return c.getTreeTargets(source, name, val)
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
	}
}
