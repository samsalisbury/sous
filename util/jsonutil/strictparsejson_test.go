package jsonutil

import (
	"strings"
	"testing"
)

func TestStrictParseJSON_structs_ok(t *testing.T) {

	type NoFields struct{}
	type OneRequiredField struct {
		A string `json:"a"`
	}
	type OneOmitemptyField struct {
		A string `json:"a,omitempty"`
	}

	cases := []struct {
		name    string
		rawJSON string
		v       interface{}
		assert  func(t *testing.T, v interface{})
	}{

		{
			name:    "empty_obj",
			v:       &NoFields{},
			rawJSON: `{}`,
		},
		{
			name:    "empty_required_field",
			v:       &OneRequiredField{},
			rawJSON: `{"a":""}`,
			assert: func(t *testing.T, v interface{}) {
				if got, want := v.(*OneRequiredField).A, ""; got != want {
					t.Errorf("got .A == %q; want %q", got, want)
				}
			},
		},
		{
			name:    "nonempty_required_field",
			v:       &OneRequiredField{},
			rawJSON: `{"a":"1"}`,
			assert: func(t *testing.T, v interface{}) {
				if got, want := v.(*OneRequiredField).A, "1"; got != want {
					t.Errorf("got .A == %q; want %q", got, want)
				}
			},
		},
		{
			name:    "missing_omitempty_field",
			v:       &OneOmitemptyField{},
			rawJSON: `{}`,
			assert: func(t *testing.T, v interface{}) {
				if got, want := v.(*OneOmitemptyField).A, ""; got != want {
					t.Errorf("got .A == %q; want %q", got, want)
				}
			},
		},
		{
			name:    "nonempty_omitempty_field",
			v:       &OneOmitemptyField{},
			rawJSON: `{"a": "1"}`,
			assert: func(t *testing.T, v interface{}) {
				if got, want := v.(*OneOmitemptyField).A, "1"; got != want {
					t.Errorf("got .A == %q; want %q", got, want)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := StrictParseJSON(c.rawJSON, c.v); err != nil {
				t.Fatal(err)
			}
			if c.assert == nil {
				return
			}
			c.assert(t, c.v)
		})
	}
}

func TestStrictParseJSON_structs_err(t *testing.T) {

	type NoFields struct{}
	type OneRequiredField struct {
		A string `json:"a"`
	}
	type OneOmitemptyField struct {
		A string `json:"a,omitempty"`
	}

	cases := []struct {
		name    string
		rawJSON string
		v       interface{}
	}{
		{
			name:    "extra_field",
			v:       &NoFields{},
			rawJSON: `{"b":"1"}`,
		},
		{
			name:    "missing_required_field",
			v:       &OneRequiredField{},
			rawJSON: `{}`,
		},
		{
			name:    "empty_omitempty_field",
			v:       &OneOmitemptyField{},
			rawJSON: `{"a":""}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			const wantErrContains = "missing or unrecognised fields"
			err := StrictParseJSON(c.rawJSON, c.v)
			if err == nil {
				t.Fatalf("got nil; want err containing %q", wantErrContains)
			}
			gotErr := err.Error()
			if !strings.Contains(gotErr, wantErrContains) {
				t.Errorf("got error %q; want it to contain %q", gotErr, wantErrContains)
			}
		})
	}
}
