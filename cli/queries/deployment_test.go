package queries

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestSimpleFilter(t *testing.T) {

	deploy := func(sid sous.SourceID) *sous.Deployment {
		return &sous.Deployment{SourceID: sid}
	}

	repoSID := func(repo string) sous.SourceID {
		return sous.SourceID{Location: sous.SourceLocation{Repo: repo}}
	}

	ds := sous.NewDeployments(
		deploy(repoSID("X")),
		deploy(repoSID("Y")),
		deploy(repoSID("Z")),
	)

	filter := simpleFilter(func(d *sous.Deployment) bool {
		return d.SourceID == repoSID("X")
	})

	trueResult := filter(ds, true)
	falseResult := filter(ds, false)

	gotTrue := trueResult.Len()
	gotFalse := falseResult.Len()

	wantTrue := 1
	wantFalse := 2
	if wantTrue+wantFalse != ds.Len() {
		t.Fatalf("bad test: wantTrue + wantFalse != total")
	}

	if gotTrue != wantTrue {
		t.Errorf("got %d true results; want %d", gotTrue, wantTrue)
	}
	if gotFalse != wantFalse {
		t.Errorf("got %d false results; want %d", gotFalse, wantFalse)
	}
}

func TestDeploymentQuery_parseFilters_ok(t *testing.T) {
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
			c := DeploymentQuery{
				StateManager: sm,
			}
			gotFilters, err := c.parseFilters(tc.filters)
			if err != nil {
				t.Fatal(err)
			}
			gotCount := len(gotFilters)
			if gotCount != tc.wantCount {
				t.Errorf("got count %d; want %d", gotCount, tc.wantCount)
			}
			for i, f := range gotFilters {
				if f == nil {
					t.Errorf("filter %d is nil", i)
				}
			}
		})
	}
}
