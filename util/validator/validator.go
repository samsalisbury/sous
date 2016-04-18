package validator

import (
	"fmt"
	"reflect"
	"strings"
)

type (
	Interface interface {
		Validate() error
	}
	validator func(reflect.Value) error
	ctx       struct {
		parent *ctx
		field  *reflect.StructField
		typ    reflect.Type
		key    bool
		index  *string
	}
	// ValidationError indicates an error with validation.
	ValidationError struct {
		ctx
		Problem string
	}
)

var validators = map[string]func(reflect.Value, *ctx) error{
	"nonempty":        nonempty,
	"keys=nonempty":   keys(nonempty),
	"values=nonempty": values(nonempty),
	"nonzero":         nonzero,
	"keys=nonzero":    keys(nonzero),
	"values=nonzero":  values(nonzero),
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s %s", e.ctx, e.Problem)
}

func (c ctx) String() string {
	if c.parent == nil {
		return c.ownString()
	}
	return c.parent.String() + "." + c.ownString()
}

func (c ctx) ownString() string {
	if c.typ != nil {
		return c.typ.String()
	}
	if c.field != nil {
		return c.field.Name
	}
	if c.key {
		return "(key)"
	}
	if c.index != nil {
		return fmt.Sprintf("[%s]", *c.index)
	}
	return "?"
}

func (c ctx) validationErrorf(format string, a ...interface{}) ValidationError {
	return ValidationError{c, fmt.Sprintf(format, a...)}
}

func (c ctx) err(err error) error {
	if ve, ok := err.(ValidationError); ok {
		ve.ctx = c
		return ve
	}
	return c.validationErrorf(err.Error())
}

func (c ctx) enterField(f reflect.StructField) ctx {
	return ctx{parent: &c, field: &f}
}

func (c ctx) enterKey() ctx {
	return ctx{parent: &c, key: true}
}

func (c ctx) enterIndex(index reflect.Value) ctx {
	i := fmt.Sprint(index.Interface())
	return ctx{parent: &c, index: &i}
}

func errIf(condition bool, format string, a ...interface{}) error {
	if !condition {
		return nil
	}
	return fmt.Errorf(format, a...)
}

func (c *ctx) validationNotPossible(which, format string, a ...interface{}) error {
	m := fmt.Sprintf(format, a...)
	return fmt.Errorf("validation rule invalid: %s `validate:%q` (%s)", c, which, m)
}

func nonempty(v reflect.Value, c *ctx) error {
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		if v.Len() != 0 {
			return nil
		}
		return c.validationErrorf("is nil or empty")
	}
	return c.validationNotPossible("nonempty", "nonempty validation not possible for %s", v.Type())
}

func nonzero(v reflect.Value, c *ctx) error {
	zero := reflect.Zero(v.Type())
	if v.Interface() == zero.Interface() {
		return c.validationErrorf("is equal to zero value (%+v)", zero.Interface())
	}
	return nil
}

func keys(f func(reflect.Value, *ctx) error) func(reflect.Value, *ctx) error {
	return func(v reflect.Value, c *ctx) error {
		if v.Kind() != reflect.Map {
			return fmt.Errorf("keys validator used on %s; only allowed on maps", v.Type())
		}
		for _, k := range v.MapKeys() {
			if err := f(k, c); err != nil {
				return c.enterKey().err(err)
			}
		}
		return nil
	}
}

func values(f func(reflect.Value, *ctx) error) func(reflect.Value, *ctx) error {
	return func(v reflect.Value, c *ctx) error {
		if v.Kind() == reflect.Map {
			return mapValues(f, v, c)
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return sliceValues(f, v, c)
		}
		return fmt.Errorf("nonemptyValues validator used on %s; only allowed on map, slice, array", v.Type())
	}
}

func mapValues(f func(reflect.Value, *ctx) error, v reflect.Value, c *ctx) error {
	for _, k := range v.MapKeys() {
		vk := v.MapIndex(k)
		if err := validateInterface(vk, c); err != nil {
			return c.enterIndex(k).err(err)
		}
		if err := f(vk, c); err != nil {
			return c.enterIndex(k).err(err)
		}
	}
	return nil
}

func sliceValues(f func(reflect.Value, *ctx) error, v reflect.Value, c *ctx) error {
	for i := 0; i < v.Len(); i++ {
		vi := v.Index(i)
		if err := validateInterface(vi, c); err != nil {
			return c.enterIndex(reflect.ValueOf(i)).err(err)
		}
		if err := f(vi, c); err != nil {
			return c.enterIndex(reflect.ValueOf(i)).err(err)
		}
	}
	return nil
}

func Validate(x interface{}) error {
	if x == nil {
		return fmt.Errorf("cannot validate nil")
	}
	v := reflect.ValueOf(x)
	c := &ctx{typ: v.Type()}
	return validateStruct(v, c)
}

func validateStruct(v reflect.Value, c *ctx) error {
	if err := validateInterface(v, c); err != nil {
		return err
	}
	k := v.Kind()
	t := v.Type()
	if k != reflect.Struct {
		if k == reflect.Ptr {
			return validateStruct(v.Elem(), c)
		}
		return fmt.Errorf("cannot validate %s (non-struct value) without context", t)
	}
	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		if err := validateStructField(v.Field(i), f, c.enterField(f)); err != nil {
			return err
		}
	}
	return nil
}

func validateInterface(v reflect.Value, c *ctx) error {
	if v.CanInterface() {
		if vi, ok := v.Interface().(Interface); ok {
			if err := vi.Validate(); err != nil {
				return c.validationErrorf(err.Error())
			}
		}
	}
	return nil
}

func validateStructField(v reflect.Value, f reflect.StructField, c ctx) error {
	// Get validators first as a separate step, so we can fail for
	// misconfiguration before failing for any validation errors.
	validators, err := getValidators(f.Tag.Get("validate"), f.Type)
	if err != nil {
		return err
	}
	for _, validate := range validators {
		if err := validate(v, &c); err != nil {
			return err
		}
	}
	return nil
}

func getValidators(tag string, typ reflect.Type) ([]func(reflect.Value, *ctx) error, error) {
	vs := []func(reflect.Value, *ctx) error{}
	tags := strings.Split(tag, ",")
	for _, tag := range tags {
		validate, ok := validators[tag]
		if !ok {
			return nil, fmt.Errorf("no validator named %q", tag)
		}
		vs = append(vs, validate)
	}
	return vs, nil
}
