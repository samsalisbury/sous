package queries

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	sous "github.com/opentable/sous/lib"
)

func repoSID(repo string) sous.SourceID {
	return sous.SourceID{Location: sous.SourceLocation{Repo: repo}}
}
func deploy(sid sous.SourceID) *sous.Deployment {
	return &sous.Deployment{SourceID: sid}
}

// assertResultCount asserts that we get want results for the filter in "true"
// mode, and that we get len(ds) - want results for filter in "false" mode.
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
	ds := sous.NewDeployments(
		deploy(repoSID("X")),
		deploy(repoSID("Y")),
		deploy(repoSID("Z")),
	)
	t.Run("0 results", func(t *testing.T) {
		filter := simpleFilter(func(d *sous.Deployment) bool {
			return d.SourceID == repoSID("?")
		})
		assertResultCount(t, ds, filter, 0)
	})
	t.Run("1 result", func(t *testing.T) {
		filter := simpleFilter(func(d *sous.Deployment) bool {
			return d.SourceID == repoSID("X")
		})
		assertResultCount(t, ds, filter, 1)
	})
	t.Run("2 results", func(t *testing.T) {
		filter := simpleFilter(func(d *sous.Deployment) bool {
			return d.SourceID != repoSID("X")
		})
		assertResultCount(t, ds, filter, 2)
	})
	t.Run("all results", func(t *testing.T) {
		filter := simpleFilter(func(d *sous.Deployment) bool {
			return true
		})
		assertResultCount(t, ds, filter, 3)
	})
}

func TestParallelFilter_ok(t *testing.T) {
	ds := sous.NewDeployments(
		deploy(repoSID("X")),
		deploy(repoSID("Y")),
		deploy(repoSID("Z")),
	)
	for maxConcurrent := 1; maxConcurrent <= 3; maxConcurrent++ {
		t.Run(fmt.Sprintf("maxConcurrent=%d", maxConcurrent), func(t *testing.T) {
			t.Run("0 results", func(t *testing.T) {
				filter := parallelFilter(maxConcurrent, func(d *sous.Deployment) (bool, error) {
					return d.SourceID == repoSID("?"), nil
				})
				assertResultCount(t, ds, filter, 0)
			})
			t.Run("1 result", func(t *testing.T) {
				filter := parallelFilter(maxConcurrent, func(d *sous.Deployment) (bool, error) {
					return d.SourceID == repoSID("X"), nil
				})
				assertResultCount(t, ds, filter, 1)
			})
			t.Run("2 results", func(t *testing.T) {
				filter := parallelFilter(maxConcurrent, func(d *sous.Deployment) (bool, error) {
					return d.SourceID != repoSID("X"), nil
				})
				assertResultCount(t, ds, filter, 2)
			})
			t.Run("all results", func(t *testing.T) {
				filter := parallelFilter(maxConcurrent, func(d *sous.Deployment) (bool, error) {
					return true, nil
				})
				assertResultCount(t, ds, filter, 3)
			})
		})
	}
}

func TestParallelFilter_err(t *testing.T) {
	ds := sous.NewDeployments(
		deploy(repoSID("X")),
		deploy(repoSID("Y")),
		deploy(repoSID("Z")),
	)

	assertErr := func(t *testing.T, filter deployFilter, wantErrContaining string) {
		t.Helper()
		_, err := filter(ds, true)
		if err == nil {
			t.Fatalf("got nil; want error containing %q", wantErrContaining)
		}
		got := err.Error()
		if !strings.Contains(got, wantErrContaining) {
			t.Errorf("got error %q; want %q", got, wantErrContaining)
		}

	}

	t.Run("zero concurrency", func(t *testing.T) {
		filter := parallelFilter(0, func(*sous.Deployment) (bool, error) {
			return true, nil // this func body is irrelevant
		})
		assertErr(t, filter, "maxConcurrent < 1 not allowed")
	})

	for maxConcurrent := 1; maxConcurrent <= 3; maxConcurrent++ {
		t.Run(fmt.Sprintf("maxConcurrent=%d", maxConcurrent), func(t *testing.T) {
			t.Run("error every time", func(t *testing.T) {
				filter := parallelFilter(maxConcurrent, func(*sous.Deployment) (bool, error) {
					return true, fmt.Errorf("always error")
				})
				assertErr(t, filter, "always error")
			})
			t.Run("error on one occasion", func(t *testing.T) {
				filter := parallelFilter(maxConcurrent, func(d *sous.Deployment) (bool, error) {
					if d.SourceID == repoSID("Y") {
						return true, fmt.Errorf("error on Y")
					}
					return true, nil
				})
				assertErr(t, filter, "error on Y")
			})
		})
	}
}

// TestParallelFilter_maxConcurrent tests that the maxConcurrent value is
// respected.
func TestParallelFilter_maxConcurrent(t *testing.T) {
	ds := sous.NewDeployments()
	for i := 0; i < 1000; i++ {
		ds.Add(deploy(repoSID(strconv.Itoa(i))))
	}
	for maxConcurrent := 1; maxConcurrent <= 10; maxConcurrent++ {
		maxConcurrent := maxConcurrent
		var current, max int
		// mu is used to synchrnonise updating current and max
		mu := sync.Mutex{}
		// blocker is used to coordinate filter func ending
		blocker := make(chan struct{})

		filterFunc := func(*sous.Deployment) (bool, error) {
			mu.Lock()
			current++
			if current > max {
				max = current
			}
			defer func() {
				mu.Lock()
				defer mu.Unlock()
				current--
			}()
			mu.Unlock()
			<-blocker
			return true, nil
		}
		// Let's try to hit max with this naughty sleep before allowing filters
		// to start returning.
		go func() {
			time.Sleep(time.Second)
			close(blocker)
		}()

		t.Run(fmt.Sprintf("maxConcurrent=%d", maxConcurrent), func(t *testing.T) {
			t.Parallel()
			pf := parallelFilter(maxConcurrent, filterFunc)
			got, err := pf(ds, true)
			if err != nil {
				t.Fatal(err)
			}
			if got.Len() != ds.Len() {
				t.Errorf("got %d results; want %d", got.Len(), ds.Len())
			}
			if max > maxConcurrent {
				t.Errorf("got %d concurrent filter funcs; want <= %d", max, maxConcurrent)
			}
			if max != maxConcurrent {
				t.Logf("NOTE: we didn't hit max concurrent; this probably isn't an issue")
			}
		})
	}

}
