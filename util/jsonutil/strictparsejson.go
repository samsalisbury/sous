package jsonutil

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// StrictParseJSON attempts to parse rawJSON into v. If any fields of the
// rawJSON are not present in v, it returns an error. If any of the fields not
// tagged omitempty are not present in the rawJSON, it returns an error.
// Fields marked omitempty are considered optional, but must not be present when
// empty in the source JSON.
// StrictParseJSON is case sensitive for field names, so you will need to use
// json field tags if rawJSON does not use UpperCaseCamel casing.
// Error messages are not very granular, "empty or missing fields" with a dump
// of the input JSON and a dump a zeroed v with required fields shown.
func StrictParseJSON(rawJSON string, v interface{}) error {
	comp := map[string]interface{}{}
	if err := json.Unmarshal([]byte(rawJSON), v); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(rawJSON), &comp); err != nil {
		return err
	}
	compJSONb, err := json.Marshal(comp)
	if err != nil {
		return err
	}
	understoodJSONb, err := json.Marshal(v)
	if err != nil {
		return err
	}
	understoodJSON := string(understoodJSONb)
	compJSON := string(compJSONb)

	equal, err := EqualJSON(compJSON, understoodJSON)
	if err != nil {
		return err
	}
	if !equal {
		return fmt.Errorf("missing or unrecognised fields:\n%swant:\n%s",
			compJSON, understoodJSON)
	}
	return nil
}

// EqualJSON compares 2 JSON strings and compares them for semantic equality.
// That is, they are both parsed to interface{} and tested for equality using
// reflect.DeepEqual. An error is returned if either a or b is invalid JSON.
func EqualJSON(a, b string) (bool, error) {
	var aVal, bVal interface{}
	if err := json.Unmarshal([]byte(a), &aVal); err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(b), &bVal); err != nil {
		return false, err
	}
	return reflect.DeepEqual(aVal, bVal), nil
}
