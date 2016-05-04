package hy

import (
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
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

func (m Marshaller) Marshal(path string, v interface{}) error {
	return ctx{path, nil, m.MarshalFunc}.marshalDir(v)
}

func (c ctx) marshalDir(v interface{}) error {
	ts, err := c.writeStructTargets(v)
	if err != nil {
		return err
	}
	ts.marshalAll(nil)
	return nil
}

func (ts targets) marshalAll(parent *reflect.Value) error {
	for _, t := range ts {
		// first marshal children
		if t.subTargets != nil {
			log.Printf("Marshalling %d children of %s", len(t.subTargets), t.val.Type())
			if err := t.subTargets.marshalAll(&t.val); err != nil {
				return err
			}
		} else {
			log.Printf("No children of %s", t.val.Type())
		}
		if parent != nil {
			log.Printf("Begin zeroing parent: %s.%s (%v)", parent.Type(), t.name, parent.Interface())
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
				log.Printf("Zeroing %s.%s (%v)\n", parent.Type(), t.name, z.Interface())
				field.Set(z)
			case reflect.Struct:
				panic("parent is struct, want *struct")
			case reflect.Map:
				z := reflect.Zero(parent.Type())
				log.Printf("Zeroing %s (%v)\n", parent.Type(), z.Interface())
				parent.Set(z)
				// do nothing, we already zero out entire maps via struct
				// field zeroing.
			}
			log.Printf("Done zeroing: %s.%s (%v)", parent.Type(), t.name, parent.Interface())
		}
		log.Printf("Final %s.%s (%v)", t.val.Type(), t.name, t.val.Interface())
		if err := t.marshal(); err != nil {
			return err
		}
	}
	return nil
}

func (t target) marshal() error {
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
	return ioutil.WriteFile(t.path, b, 0666)
}
