package queries

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestDeployment_ParseAttributeFilters_ok(t *testing.T) {
	cases := []struct {
		filters   string
		wantCount int
	}{
		{"", 0},
		{"hasimage=true", 1},
		{"hasimage=false", 1},
		{"hasowners=true", 1},
		{"hasowners=false", 1},
		{"zeroinstances=true", 1},
		{"zeroinstances=false", 1},
		{"zeroinstances=false hasowners=true", 2},
		{"zeroinstances=false hasowners=true hasimage=false", 3},
	}
	for _, tc := range cases {
		t.Run(tc.filters, func(t *testing.T) {
			sm := sous.NewDummyStateManager()
			c := Deployment{
				StateManager: sm,
			}
			got, err := c.ParseAttributeFilters(tc.filters)
			if err != nil {
				t.Fatal(err)
			}
			gotCount := len(got.filters)
			if gotCount != tc.wantCount {
				t.Errorf("got count %d; want %d", gotCount, tc.wantCount)
			}
			for i, f := range got.filters {
				if f == nil {
					t.Errorf("filter %d is nil", i)
				}
			}
		})
	}
}

func TestDeployment_Result(t *testing.T) {
	sm := sous.NewDummyStateManager()
	sm.State = sous.DefaultStateFixture()
	aq := ArtifactQuery{}
	q := Deployment{
		StateManager:  sm,
		ArtifactQuery: aq,
	}
	af, err := q.ParseAttributeFilters("")
	if err != nil {
		t.Fatal(err)
	}
	r, err := q.Result(DeploymentFilters{AttributeFilters: af})
	if err != nil {
		t.Fatal(err)
	}
	want := 9 // NOTE SS: sous.DefaultStateFixture returns 9 deployments as standard.
	got := r.Deployments.Len()
	if got != want {
		t.Errorf("got %d deployments; want %d", got, want)
	}
}
