package validator

import "testing"

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
	NonzeroStructMapKey struct {
		Map map[Struct]Struct `validate:"keys=nonzero"`
	}
	NonzeroStructMapValue struct {
		Map map[Struct]Struct `validate:"values=nonzero"`
	}
)

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
			"validator.NonZeroStruct.Struct is equal to zero value ({String: Int:0})"},
		{NonZeroStruct{Struct{String: ""}},
			"validator.NonZeroStruct.Struct is equal to zero value ({String: Int:0})"},
		{NonZeroStruct{Struct{Int: 0}},
			"validator.NonZeroStruct.Struct is equal to zero value ({String: Int:0})"},
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

func TestValidate_NonZeroStruct(t *testing.T) {
	shouldBeValid := []NonZeroStruct{
		{Struct{String: "x"}},
		{Struct{Int: 1}},
	}
	for _, x := range shouldBeValid {
		if err := Validate(x); err != nil {
			t.Errorf("unexpected error %q for %+v", err, x)
		}
	}
}

func TestValidate_NonemptyStringMapKey(t *testing.T) {
	shouldBeValid := []NonemptyStringMapKey{
		{Map: nil},
		{Map: map[string]string{"x": ""}},
	}
	for _, x := range shouldBeValid {
		if err := Validate(x); err != nil {
			t.Errorf("unexpected error %q for %+v", err, x)
		}
	}
}

func TestValidate_NonemptyStringMapVal(t *testing.T) {
	shouldBeValid := []NonemptyStringMapVal{
		{Map: nil},
		{Map: map[string]string{"": "x"}},
	}
	for _, x := range shouldBeValid {
		if err := Validate(x); err != nil {
			t.Errorf("unexpected error %q for %+v", err, x)
		}
	}
}

func TestValidate_NonemptyString(t *testing.T) {
	x := NonemptyString{"x"}
	if err := Validate(x); err != nil {
		t.Errorf("unexpected error %q", err)
	}
}

func TestValidate_NonemptyMap(t *testing.T) {
	shouldBeValid := []NonemptyMap{
		{Map: map[string]string{"": ""}},
		{Map: map[string]string{"": "x"}},
		{Map: map[string]string{"x": ""}},
	}
	for _, x := range shouldBeValid {
		if err := Validate(x); err != nil {
			t.Errorf("unexpected error %q", err)
		}
	}
}

func TestValidate_NonemptySlice(t *testing.T) {
	shouldBeValid := []NonemptySlice{
		{Slice: []string{""}},
		{Slice: []string{"hi"}},
	}
	for _, x := range shouldBeValid {
		x.Slice = []string{""}
		if err := Validate(x); err != nil {
			t.Errorf("unexpected error %q", err)
		}
	}
}
