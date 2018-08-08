package sous

import (
	"reflect"
	"testing"

	"github.com/samsalisbury/semv"
)

func TestDeploySpec_ClusterNames(t *testing.T) {
	cases := []struct {
		in   DeploySpecs
		want []string
	}{
		{in: DeploySpecs{"a": DeploySpec{}}, want: []string{"a"}},
		{in: DeploySpecs{"a": DeploySpec{}, "b": DeploySpec{}}, want: []string{"a", "b"}},
		{in: DeploySpecs{"b": DeploySpec{}, "a": DeploySpec{}}, want: []string{"a", "b"}},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			got := c.in.ClusterNames()
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v; want %v", got, c.want)
			}
		})
	}
}

func TestDeploySpec_Validate(t *testing.T) {
	invalidConfig := DeployConfig{}
	dcFlaws := invalidConfig.Validate()
	if len(dcFlaws) == 0 {
		t.Fatalf("test setup failed to produce invalid DeployConfig")
	}

	invalidSpec := DeploySpec{DeployConfig: invalidConfig}
	specFlaws := invalidSpec.Validate()
	if len(specFlaws) < len(dcFlaws) {
		t.Fatalf("validating DeploySpec produced %d flaws; want at least %d",
			len(specFlaws), len(dcFlaws))
	}

	// NOTE SS: Flaws are differentiated by type, assertions on actual value
	// are difficult.
	for i, wantFlaw := range dcFlaws {
		gotFlaw := specFlaws[i]
		got, want := reflect.TypeOf(gotFlaw), reflect.TypeOf(wantFlaw)
		if got != want {
			t.Errorf("flaws do not match, got %d:%s; want %s", i, got, want)
		}
	}
}

func TestDeploySpec_Equal(t *testing.T) {
	cases := []struct {
		a, b DeploySpec
		want bool
	}{
		{a: DeploySpec{}, b: DeploySpec{}, want: true},
		{a: DeploySpec{Version: semv.MustParse("1")}, b: DeploySpec{}, want: false},
	}

	for _, c := range cases {
		got := c.a.Equal(c.b)
		if got != c.want {
			t.Fatalf("got %v.Equal(%v) == %t; want %t", c.a, c.b, got, c.want)
		}

		gotReverse := c.b.Equal(c.a)
		if got != gotReverse {
			t.Errorf("Equal method is not transverse")
		}
	}
}
