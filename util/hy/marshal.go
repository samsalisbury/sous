package hy

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/opentable/sous/util/yaml"
)

type (
	Marshaller struct {
		MarshalFunc func(interface{}) ([]byte, error)
	}
)

func NewMarshaller(marshalFunc func(interface{}) ([]byte, error)) Marshaller {
	if marshalFunc == nil {
		panic("marshalFunc cannot be nil")
	}
	return Marshaller{marshalFunc}
}

// Marshal is shorthand for NewMarshaller(yaml.Marshal).Marshal
func Marshal(dir string, v interface{}) error {
	return NewMarshaller(yaml.Marshal).Marshal(dir, v)
}

func (m Marshaller) Marshal(path string, v interface{}) error {
	return ctx{path, nil, m.MarshalFunc}.marshalDir(v)
}

func (c ctx) marshalDir(v interface{}) error {
	ts, err := c.writeStructTargets(v)
	if err != nil {
		return err
	}
	return ts.marshalAll(nil)
}

func (ts targets) marshalAll(parent *reflect.Value) error {
	for _, t := range ts {
		t.marshal(parent)
	}
	return nil
}

func (t target) marshal(parent *reflect.Value) error {
	// first marshal children
	if len(t.subTargets) != 0 {
		debugf("Marshalling %d children of %s", len(t.subTargets), t.val.Type())
		if err := t.subTargets.marshalAll(&t.val); err != nil {
			return err
		}
	} else {
		debugf("No children of %s", t.val.Type())
	}
	if parent != nil {
		debugf("Begin zeroing parent: %s.%s (%v)", parent.Type(), t.name, parent.Interface())
		// zero out the field in the parent
		switch parent.Kind() {
		default:
			return fmt.Errorf("parents may only be structs or map[string]T")
		case reflect.Ptr:
			field := parent.Elem().FieldByName(t.name)
			if !field.CanSet() {
				return fmt.Errorf("unable to set %s on %s", t.name, parent.Type())
			}
			z := reflect.Zero(field.Type())
			debugf("Zeroing %s.%s (%v)\n", parent.Type(), t.name, z.Interface())
			field.Set(z)
		case reflect.Struct:
			panic("parent is struct, want *struct or map")
		case reflect.Map:
			z := reflect.Zero(parent.Type())
			debugf("Zeroing %s (%v)\n", parent.Type(), z.Interface())
			parent.Set(z)
			// do nothing, we already zero out entire maps via struct
			// field zeroing.
		}
		debugf("Done zeroing: %s.%s (%v)", parent.Type(), t.name, parent.Interface())
	}
	if t.typ.Kind() == reflect.Map {
		debugf("Not writing map %s (%s)", t.name, t.typ)
		return nil
	}
	debugf("Final %s.%s (%v)", t.val.Type(), t.name, t.val.Interface())
	return t.write()
}

func (t target) write() error {
	// first convert to map[string]interface{}, then delete keys that have
	// hy tags, and finally marshal what's left.
	// this will cause issues with field name mappings used by the marshaller/
	// unmarshaller. This could potentially be avoided by first marshalling
	// everything, then unmarshalling to map and deleting.
	if t.marshalFunc == nil {
		panic("marshalFunc is nil")
	}
	b, err := t.marshalFunc(t.val.Interface())
	if err != nil {
		return err
	}
	dir := path.Dir(t.path)
	if err := ensureDirExists(dir); err != nil {
		return err
	}
	path := t.path
	if !strings.HasSuffix(path, ".yaml") {
		path += ".yaml"
	}
	debug("Writing file", path, string(b))
	return ioutil.WriteFile(path, b, 0777)
}

func ensureDirExists(path string) error {
	d, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, 0777)
		}
		return err
	}
	if d.IsDir() {
		return nil
	}
	return fmt.Errorf("%s exists and is not a directory", path)
}
