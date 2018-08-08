package sous

import (
	"reflect"
	"testing"
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
