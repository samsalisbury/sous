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
		parent                                *ctx
		field                                 *reflect.StructField
		keyVal, valueVal, fieldVal, structVal *reflect.Value
	}
	// ValidationError indicates an error with validation.
	ValidationError struct {
		ctx
		Problem string
	}
)

var validators map[string]func(reflect.Value, ctx) error

func init() {
	validators = map[string]func(reflect.Value, ctx) error{
		"nonempty":        nonempty,
		"keys=nonempty":   keys(nonempty),
		"values=nonempty": values(nonempty),
		"nonzero":         nonzero,
		"keys=nonzero":    keys(nonzero),
		"values=nonzero":  values(nonzero),
	}
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
	if c.keyVal != nil {
		if c.valueVal == nil {
			return "(key)"
		} else {
			return fmt.Sprintf("[%+v]", c.keyVal.Interface())
		}
	}
	if c.structVal != nil {
		return c.structVal.Type().String()
	}
	if c.field != nil {
		return c.field.Name
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

func (c ctx) enterStruct(v reflect.Value) ctx {
	return ctx{parent: nil, structVal: &v}
}

func (c ctx) enterField(v reflect.Value, f reflect.StructField) ctx {
	return ctx{parent: &c, fieldVal: &v, field: &f}
}

func (c ctx) enterKey(key reflect.Value) ctx {
	return ctx{parent: &c, keyVal: &key, field: c.field}
}

func (c ctx) enterKeyValue(key, value reflect.Value) ctx {
	return ctx{parent: &c, keyVal: &key, valueVal: &value, field: c.field}
}

func (c ctx) validationNotPossible(which, format string, a ...interface{}) error {
	m := fmt.Sprintf(format, a...)
	return fmt.Errorf("validation rule invalid: %s `validate:%q` (%s)", c, which, m)
}

func nonempty(v reflect.Value, c ctx) error {
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		if v.Len() != 0 {
			return nil
		}
		return c.validationErrorf("is nil or empty")
	}
	return c.validationNotPossible("nonempty", "nonempty validation not possible for %s", v.Type())
}

func canBeNil(v reflect.Value) bool {
	switch v.Kind() {
	default:
		return false
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
}

func safeIsNil(v reflect.Value) bool {
	return canBeNil(v) && v.IsNil()
}

func nonzero(v reflect.Value, c ctx) error {
	if safeIsNil(v) {
		return c.validationErrorf("is equal to its zero value (nil)")
	}
	zero := reflect.Zero(v.Type())
	if reflect.DeepEqual(v.Interface(), zero.Interface()) {
		return c.validationErrorf("is equal to its zero value (%+v)", zero.Interface())
	}
	return nil
}

// keys creates a validator that validates each key in a map, slice, or array
func keys(f func(reflect.Value, ctx) error) func(reflect.Value, ctx) error {
	return func(v reflect.Value, c ctx) error {
		if v.Kind() != reflect.Map {
			return fmt.Errorf("keys validator used on %s; only allowed on maps", v.Type())
		}
		for _, vk := range v.MapKeys() {
			c := c.enterKey(vk)
			if err := c.validateInterface(vk); err != nil {
				return err
			}
			if err := f(vk, c); err != nil {
				return err
			}
		}
		return nil
	}
}

// values creates a validator that validates each value in a map, slice, or
// array
func values(f func(reflect.Value, ctx) error) func(reflect.Value, ctx) error {
	return func(v reflect.Value, c ctx) error {
		if v.Kind() == reflect.Map {
			return mapValues(f, v, c)
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return sliceValues(f, v, c)
		}
		return fmt.Errorf("nonemptyValues validator used on %s; only allowed on map, slice, array", v.Type())
	}
}

func mapValues(f func(reflect.Value, ctx) error, v reflect.Value, c ctx) error {
	for _, vk := range v.MapKeys() {
		k := v.MapIndex(vk)
		c := c.enterKeyValue(vk, k)
		if err := c.validateInterface(k); err != nil {
			return err
		}
		if err := f(k, c); err != nil {
			return err
		}
	}
	return nil
}

func sliceValues(f func(reflect.Value, ctx) error, v reflect.Value, c ctx) error {
	for i := 0; i < v.Len(); i++ {
		vi := reflect.ValueOf(i)
		v := v.Index(i)
		c := c.enterKeyValue(vi, v)
		if err := c.validateInterface(v); err != nil {
			return err
		}
		if err := f(vi, c); err != nil {
			return err
		}
	}
	return nil
}

func Validate(x interface{}) error {
	if x == nil {
		return fmt.Errorf("cannot validate nil")
	}
	return (&ctx{}).enterStruct(reflect.ValueOf(x)).validate()
}

func (c ctx) validate() error {
	if c.fieldVal != nil {
		return c.validateStructField()
	}
	if c.structVal != nil {
		return c.validateStruct()
	}
	panic("neither index, value, field, typ set")
}

func (c ctx) validateStruct() error {
	v := *c.structVal
	if err := c.validateInterface(v); err != nil {
		return err
	}
	k := v.Kind()
	t := v.Type()
	if k != reflect.Struct {
		if k == reflect.Ptr {
			e := v.Elem()
			c.structVal = &e
			return c.validateStruct()
		}
		return fmt.Errorf("cannot validate %s (non-struct value) without context", t)
	}
	for i := 0; i < v.NumField(); i++ {
		if err := c.enterField(v.Field(i), t.Field(i)).validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateStruct(v reflect.Value, c ctx) error {
	k := v.Kind()
	if k == reflect.Ptr && !v.IsNil() {
		return validateStruct(v.Elem(), c)
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	return c.enterStruct(v).validate()
}

func (c ctx) validateStructField() error {
	v := *c.fieldVal
	f := c.field
	// Get validators first as a separate step, so we can fail for
	// misconfiguration before failing for any validation errors.
	validators, err := getValidators(*f)
	if err != nil {
		return err
	}
	for _, validate := range validators {
		if err := validate(v, c); err != nil {
			return err
		}
	}
	return nil
}

func (c ctx) validateInterface(v reflect.Value) error {
	if !v.CanInterface() {
		return fmt.Errorf("unable to interface %s", v.Type())
	}
	vi, implementsInterface := v.Interface().(Interface)
	if !implementsInterface {
		return nil
	}
	if err := vi.Validate(); err != nil {
		return c.validationErrorf(err.Error())
	}
	return nil
}

func validateInterface(v reflect.Value, c ctx) error {
	return c.validateInterface(v)
}

func canValidateAsStruct(t reflect.Type) bool {
	k := t.Kind()
	r := k == reflect.Struct || (k == reflect.Ptr && t.Elem().Kind() == reflect.Struct)
	return r
}

func getValidators(f reflect.StructField) ([]func(reflect.Value, ctx) error, error) {
	// Add defaulf validators (Interface and struct) to this field.
	vs := []func(reflect.Value, ctx) error{validateInterface}
	if canValidateAsStruct(f.Type) {
		vs = append(vs, validateStruct)
	}
	k := f.Type.Kind()
	// For maps, add default validators to their key values.
	if k == reflect.Map {
		vs = append(vs, keys(validateInterface))
		if canValidateAsStruct(f.Type.Key()) {
			vs = append(vs, keys(validateStruct))
		}
	}
	// For map, slice and array, add default validators to their item values.
	if k == reflect.Map || k == reflect.Slice || k == reflect.Array {
		vs = append(vs, values(validateInterface))
		if canValidateAsStruct(f.Type.Elem()) {
			vs = append(vs, values(validateStruct))
		}
	}
	tag := f.Tag.Get("validate")
	if len(tag) == 0 {
		return vs, nil
	}
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
