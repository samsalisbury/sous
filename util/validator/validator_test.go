package validator

import (
	"fmt"
	"testing"
)

type (
	inout struct {
		in  interface{}
		out string
	}
	NonemptyString struct {
		String string `validate:"nonempty"`
	}
	NonemptyMap struct {
		Map map[string]string `validate:"nonempty"`
	}
	NonemptySlice struct {
		Slice []string `validate:"nonempty"`
	}
	// InvalidNonemptyInt cannot be validated since ints do not support nonempty.
	InvalidNonemptyInt struct {
		Int int `validate:"nonempty"`
	}
	NonemptyStringMapKey struct {
		Map map[string]string `validate:"keys=nonempty"`
	}
	NonemptyStringMapVal struct {
		Map map[string]string `validate:"values=nonempty"`
	}
	NonZeroStruct struct {
		Struct `validate:"nonzero"`
	}
	Struct struct {
		String string
		Int    int
	}
	NonZeroStructMapKey struct {
		Map map[Struct]Struct `validate:"keys=nonzero"`
	}
	NonZeroStructMapValue struct {
		Map map[Struct]Struct `validate:"values=nonzero"`
	}
	InterfaceInvalidKey struct {
		Map map[Name]InterfaceInvalidKey
	}
	InterfaceInvalidValue struct {
		Map map[string]Name
	}
	InterfaceInvalidField struct {
		Name Name
	}
	Name string
	// NestedStructs can never be valid in finite space, since it's recursive.
	// Don't really make structs like this!
	NestedStructs struct {
		Map   map[Name]*NestedStructs `validate:"values=nonzero"`
		Other *NestedStructs
	}
)

func (n Name) Validate() error {
	if len(n) > 3 {
		return fmt.Errorf("(%T(%s)) is too big; must be less than 3 characters", n, n)
	}
	return nil
}

func TestValidate_NestedStructs(t *testing.T) {
	nested := NestedStructs{
		Map: map[Name]*NestedStructs{"hi": &NestedStructs{
			Other: &NestedStructs{
				Map: map[Name]*NestedStructs{"ho": &NestedStructs{
					Map: map[Name]*NestedStructs{"its": nil}}}}}}}

	err := Validate(nested)

	if err == nil {
		t.Fatalf("%T (%+v) should have failed validation", nested, nested)
	}
	expected := "validator.NestedStructs.Map.[its] is equal to its zero value (nil)"
	actual := err.Error()
	if actual != expected {
		t.Errorf("got %q for %+v; want %q", actual, nested, expected)
	}
}

func TestValidate_Interface_InvalidKey(t *testing.T) {
	iik := InterfaceInvalidKey{Map: map[Name]InterfaceInvalidKey{Name("hello"): InterfaceInvalidKey{}}}
	err := Validate(iik)
	if err == nil {
		t.Fatalf("%T (%+v) should have failed validation", iik, iik)
	}
	actual := err.Error()
	expected := fmt.Sprintf(`%T.Map.(key) (%T(hello)) is too big; must be less than 3 characters`, iik, Name(""))
	if actual != expected {
		t.Errorf("got %q for %+v; want %q", actual, iik, expected)
	}
}

func TestValidate_Interface_InvalidValue(t *testing.T) {
	iiv := InterfaceInvalidValue{Map: map[string]Name{"hello": Name("toolong")}}
	err := Validate(iiv)
	if err == nil {
		t.Fatalf("%T (%+v) should have failed validation", iiv, iiv)
	}
	actual := err.Error()
	expected := fmt.Sprintf(`%T.Map.[hello] (%T(toolong)) is too big; must be less than 3 characters`, iiv, Name(""))
	if actual != expected {
		t.Errorf("got %q for %+v; want %q", actual, iiv, expected)
	}
}

func TestValidate_Interface_InvalidField(t *testing.T) {
	iif := InterfaceInvalidField{Name("toolong")}
	err := Validate(iif)
	if err == nil {
		t.Fatalf("%T (%+v) should have failed validation", iif, iif)
	}
	actual := err.Error()
	expected := fmt.Sprintf(`%T.Name (%T(toolong)) is too big; must be less than 3 characters`, iif, Name(""))
	if actual != expected {
		t.Errorf("got %q for %+v; want %q", actual, iif, expected)
	}
}

func TestValidate_Invalid(t *testing.T) {
	invalid := []inout{
		{NonemptyString{},
			"validator.NonemptyString.String is nil or empty"},
		{NonemptyMap{Map: nil},
			"validator.NonemptyMap.Map is nil or empty"},
		{NonemptyMap{Map: map[string]string{}},
			"validator.NonemptyMap.Map is nil or empty"},
		{NonemptySlice{Slice: nil},
			"validator.NonemptySlice.Slice is nil or empty"},
		{NonemptySlice{Slice: []string{}},
			"validator.NonemptySlice.Slice is nil or empty"},
		{NonemptyStringMapKey{Map: map[string]string{"": ""}},
			"validator.NonemptyStringMapKey.Map.(key) is nil or empty"},
		{NonemptyStringMapKey{Map: map[string]string{"": "x"}},
			"validator.NonemptyStringMapKey.Map.(key) is nil or empty"},
		{NonemptyStringMapVal{Map: map[string]string{"": ""}},
			"validator.NonemptyStringMapVal.Map.[] is nil or empty"},
		{NonemptyStringMapVal{Map: map[string]string{"x": ""}},
			"validator.NonemptyStringMapVal.Map.[x] is nil or empty"},
		{NonZeroStruct{},
			"validator.NonZeroStruct.Struct is equal to its zero value ({String: Int:0})"},
		{NonZeroStruct{Struct{String: ""}},
			"validator.NonZeroStruct.Struct is equal to its zero value ({String: Int:0})"},
		{NonZeroStruct{Struct{Int: 0}},
			"validator.NonZeroStruct.Struct is equal to its zero value ({String: Int:0})"},
		{NonZeroStructMapKey{map[Struct]Struct{Struct{}: Struct{}}},
			"validator.NonZeroStructMapKey.Map.(key) is equal to its zero value ({String: Int:0})"},
		{NonZeroStructMapKey{map[Struct]Struct{Struct{}: Struct{"x", 1}}},
			"validator.NonZeroStructMapKey.Map.(key) is equal to its zero value ({String: Int:0})"},
		{NonZeroStructMapValue{map[Struct]Struct{Struct{}: Struct{}}},
			"validator.NonZeroStructMapValue.Map.[{String: Int:0}] is equal to its zero value ({String: Int:0})"},
		{NonZeroStructMapValue{map[Struct]Struct{Struct{"x", 1}: Struct{}}},
			"validator.NonZeroStructMapValue.Map.[{String:x Int:1}] is equal to its zero value ({String: Int:0})"},
		{InvalidNonemptyInt{Int: 80085},
			"validation rule invalid: validator.InvalidNonemptyInt.Int `validate:\"nonempty\"` (nonempty validation not possible for int)"},
	}
	for _, pair := range invalid {
		x := pair.in
		expected := pair.out
		err := Validate(x)
		if err == nil {
			t.Errorf("%+v unexpectedly reported as valid", x)
			continue
		}
		actual := err.Error()
		if actual != expected {
			t.Errorf("got %q for %+v; want %q", actual, x, expected)
		}
	}
}

func TestValidate_Valid(t *testing.T) {
	valid := []interface{}{
		NonemptyString{"x"},
		NonZeroStruct{Struct{String: "x"}},
		NonZeroStruct{Struct{Int: 1}},
		NonemptyMap{Map: map[string]string{"": ""}},
		NonemptyMap{Map: map[string]string{"": "x"}},
		NonemptyMap{Map: map[string]string{"x": ""}},
		NonemptySlice{Slice: []string{""}},
		NonemptySlice{Slice: []string{"hi"}},
		NonemptyStringMapKey{Map: nil},
		NonemptyStringMapKey{Map: map[string]string{"x": ""}},
		NonemptyStringMapVal{Map: nil},
		NonemptyStringMapVal{Map: map[string]string{"": "x"}},
	}
	for _, x := range valid {
		if err := Validate(x); err != nil {
			t.Errorf("unexpected error %q for %+v", err, x)
		}
	}
}
