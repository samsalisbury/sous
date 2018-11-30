package queries

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func repoSID(repo string) sous.SourceID {
	return sous.SourceID{Location: sous.SourceLocation{Repo: repo}}
}
func deploy(sid sous.SourceID) *sous.Deployment {
	return &sous.Deployment{SourceID: sid}
}

func assertSingleMatchingResult(t *testing.T, filter deployFilter) {
	ds := sous.NewDeployments(
		deploy(repoSID("X")),
		deploy(repoSID("Y")),
		deploy(repoSID("Z")),
	)
	assertResultCount(t, ds, filter, 1)
}

func assertResultCount(t *testing.T, ds sous.Deployments, filter deployFilter, want int) {
	t.Helper()

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

	wantTrue := want
	wantFalse := ds.Len() - want

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
