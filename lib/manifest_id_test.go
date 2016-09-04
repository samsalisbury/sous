package sous

import (
	"fmt"
	"reflect"
	"testing"
)

var manifestIDTests = []struct {
	String string
	MID    ManifestID
}{
	{
		String: "github.com/user/repo",
		MID: ManifestID{
			Source: SourceLocation{
				Repo: "github.com/user/repo",
			},
		},
	},
	{
		String: "github.com/user/repo,some-dir",
		MID: ManifestID{
			Source: SourceLocation{
				Repo: "github.com/user/repo",
				Dir:  "some-dir",
			},
		},
	},
	{
		String: "github.com/user/repo,some-dir:british-flavoured",
		MID: ManifestID{
			Source: SourceLocation{
				Repo: "github.com/user/repo",
				Dir:  "some-dir",
			},
			Flavor: "british-flavoured",
		},
	},
}

func TestParseManifestID(t *testing.T) {
	for _, test := range manifestIDTests {
		input := test.String
		expected := test.MID
		actual, err := ParseManifestID(input)
		if err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("%q got %#v; want %#v", input, actual, expected)
		}
	}
}

func TestManifestID_String(t *testing.T) {
	for _, test := range manifestIDTests {
		expected := test.String
		actual := test.MID.String()
		if actual != expected {
			t.Errorf("%#v got %q; want %q", test.MID, actual, expected)
		}
	}
}

func TestManifestID_MarshalText(t *testing.T) {
	for _, test := range manifestIDTests {
		input := test.MID
		expected := test.String
		actualBytes, err := input.MarshalText()
		if err != nil {
			t.Error(err)
			continue
		}
		actual := string(actualBytes)
		if actual != expected {
			t.Errorf("%#v got %q; want %q", input, actual, expected)
		}
	}
}

func TestManifestID_MarshalYAML(t *testing.T) {
	for _, test := range manifestIDTests {
		input := test.MID
		expected := test.String
		actualInterface, err := input.MarshalYAML()
		if err != nil {
			t.Error(err)
			continue
		}
		actual := actualInterface.(string)
		if actual != expected {
			t.Errorf("%#v got %q; want %q", input, actual, expected)
		}
	}
}

func TestManifestID_UnmarshalText(t *testing.T) {
	for _, test := range manifestIDTests {
		input := []byte(test.String)
		expected := test.MID
		var actual ManifestID
		if err := actual.UnmarshalText(input); err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("%q got %#v; want %#v", input, actual, expected)
		}
	}
}

func TestManifestID_UnmarshalYAML(t *testing.T) {
	for _, test := range manifestIDTests {
		inputStr := test.String
		expected := test.MID
		var actual ManifestID
		// inputFunc mocks out behaviour of the github.com/go-yaml/yaml library.
		inputFunc := func(v interface{}) error {
			sp := reflect.ValueOf(v)
			if sp.Kind() != reflect.Ptr {
				return fmt.Errorf("got %s; want *string", sp.Type())
			}
			s := sp.Elem()
			if s.Kind() != reflect.String {
				return fmt.Errorf("got %s; want a *string", sp.Type())
			}
			s.Set(reflect.ValueOf(inputStr))
			return nil
		}
		if err := actual.UnmarshalYAML(inputFunc); err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("%q got %#v; want %#v", inputStr, actual, expected)
		}
	}
}
