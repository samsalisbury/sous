package queries

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func repoSID(repo string) sous.SourceID {
	return sous.SourceID{Location: sous.SourceLocation{Repo: repo}}
}

func assertSingleMatchingResult(t *testing.T, filter deployFilter) {
	t.Helper()
	deploy := func(sid sous.SourceID) *sous.Deployment {
		return &sous.Deployment{SourceID: sid}
	}

	ds := sous.NewDeployments(
		deploy(repoSID("X")),
		deploy(repoSID("Y")),
		deploy(repoSID("Z")),
	)

	trueResult, trueErr := filter(ds, true)
	falseResult, falseErr := filter(ds, false)

	if trueErr != nil {
		t.Fatal(trueErr)
	}
	if falseErr != nil {
		t.Fatal(falseErr)
	}

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

func TestSimpleFilter(t *testing.T) {
	filter := simpleFilter(func(d *sous.Deployment) bool {
		return d.SourceID == repoSID("X")
	})
	assertSingleMatchingResult(t, filter)
}

func TestParallelFilter_ok(t *testing.T) {
	filter := parallelFilter(1, func(d *sous.Deployment) (bool, error) {
		return d.SourceID == repoSID("X"), nil
	})
	assertSingleMatchingResult(t, filter)
}
