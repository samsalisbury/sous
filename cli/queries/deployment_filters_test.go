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
