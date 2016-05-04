package hy

import "reflect"

type (
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
		marshalFunc   func(interface{}) ([]byte, error)
	}
	targets []*target
)
