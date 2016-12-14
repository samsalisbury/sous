package otpl

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestSingularityResources_SousResources(t *testing.T) {
	tests := []struct {
		Singularity SingularityResources
		Sous        sous.Resources
	}{
		{ // This won't really happen.
			SingularityResources{
				"cpu":    1,
				"ports":  1,
				"memory": 1,
			},
			sous.Resources{
				"cpu":    "1",
				"ports":  "1",
				"memory": "1",
			},
		},
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
