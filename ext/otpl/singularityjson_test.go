package otpl

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestParseSingularityJSON_ok(t *testing.T) {

	in := `
	{
		"requestId": "anything",
		"resources": {
			"numPorts": 1,
			"memoryMb": 1,
			"cpus": 1
		},
		"env": {
			"ENV_1": "val 1"
		}
	}`

	want := SingularityJSON{
		RequestID: "anything",
		Resources: SingularityResources{
			"numPorts": 1,
			"memoryMb": 1,
			"cpus":     1,
		},
		Env: sous.Env{
			"ENV_1": "val 1",
		},
	}

	got, err := parseSingularityJSON(in)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got -->\n% #v\nwant -->\n% #v", got, want)
	}
}

func TestParseSingularityJSON_err_fields(t *testing.T) {

	cases := []string{
		`{"invalid": {}}`,
		`{"env": {"ENV_1": "val 1"}, "invalid": {}}`,
		`
		{
			"requestId": "anything",
			"resources": {
				"numPorts": 1,
				"memoryMb": 1,
				"cpus": 1
			},
			"env": {
				"ENV_1": "val 1"
			},
			"something_invalid": "yes"
		}`,
	}

	const wantPrefix = `missing or unrecognised fields:`

	for i, in := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, gotErr := parseSingularityJSON(in)
			if gotErr == nil {
				t.Fatalf("got nil error; want error beginning %q", wantPrefix)
			}
			got := gotErr.Error()
			if !strings.HasPrefix(got, wantPrefix) {
				t.Errorf("got %q; want string with prefix %q", got, wantPrefix)
			}
		})
	}
}

func TestParseSingularityRequestJSON_ok(t *testing.T) {

	in := `
	{
		"id": "anything",
		"instances": 1,
		"owners": ["owner1@example.com"]
	}`

	want := SingularityRequestJSON{
		ID:        "anything",
		Instances: 1,
		Owners:    []string{"owner1@example.com"},
	}

	got, err := parseSingularityRequestJSON(in)

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got -->\n% #v\nwant -->\n% #v", got, want)
	}
}

func TestParseSingularityRequestJSON_err_fields(t *testing.T) {

	cases := []string{
		`{"invalid": {}}`,
		`{"env": {"ENV_1": "val 1"}, "invalid": {}}`,
		`
		{
			"id": "anything",
			"instances": 1,
			"owners": ["owner1@example.com"],
			"something_invalid": "yes"
		}`,
	}

	const wantPrefix = `unrecognised fields:`

	for i, in := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, gotErr := parseSingularityRequestJSON(in)
			if gotErr == nil {
				t.Fatalf("got nil error; want error beginning %q", wantPrefix)
			}
			got := gotErr.Error()
			if !strings.HasPrefix(got, wantPrefix) {
				t.Errorf("got %q; want string with prefix %q", got, wantPrefix)
			}
		})
	}
}

func TestParseSingularityJSON_invalidResources(t *testing.T) {

	cases := []struct {
		in, err string
	}{
		{
			in: `
			{
				"requestId": "anything",
				"resources": {"numPorts": 1,"memoryMb": 1,"cpus": 1,"blah": 1}
			}`,
			err: `invalid resource name "blah"`,
		},
		{
			in: `
			{
				"requestId": "anything",
				"resources": {"numPorts": 1,"memoryMb": 1}
			}`,
			err: `missing resource(s): cpus`,
		},
		{
			in: `
			{
				"requestId": "anything",
				"resources": {"numPorts": 1,"cpus": 1}
			}`,
			err: `missing resource(s): memoryMb`,
		},
		{
			in: `
			{
				"requestId": "anything",
				"resources": {"memoryMb": 1,"cpus": 1}
			}`,
			err: `missing resource(s): numPorts`,
		},
		{
			in: `
			{
				"requestId": "anything",
				"resources": {"memoryMb": 1}
			}`,
			err: `missing resource(s): cpus, numPorts`,
		},
		{
			in: `
			{
				"requestId": "anything",
				"resources": {}
			}`,
			err: `missing resource(s): cpus, memoryMb, numPorts`,
		},
	}

	for _, c := range cases {
		t.Run(c.err, func(t *testing.T) {
			in, want := c.in, c.err
			_, gotErr := parseSingularityJSON(in)
			if gotErr == nil {
				t.Fatalf("got nil error; want %q", want)
			}
			got := gotErr.Error()
			if got != want {
				t.Errorf("got %q; want %q", got, want)
			}
		})
	}
}

func TestSingularityResources_SousResources(t *testing.T) {
	tests := []struct {
		Singularity SingularityResources
		Sous        sous.Resources
	}{
		{ // Mapping singularity resource names to Sous ones.
			SingularityResources{
				"cpu":      1,
				"numPorts": 1,
				"memoryMb": 1,
			},
			sous.Resources{
				"cpu":    "1",
				"ports":  "1",
				"memory": "1",
			},
		},
	}

	for i, test := range tests {
		input := test.Singularity
		expected := test.Sous

		actual := input.SousResources()
		if !actual.Equal(expected) {
			t.Errorf("got resources %# v; want %# v; for input %d %# v",
				actual, expected, i, input)
		}
	}
}
