package testmatrix

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

// Supervisor supervises a set of Runners, and collates their results.
// There should be exactly one global Supervisor in every package that uses
// testmatrix.
type Supervisor struct {
	mu             sync.Mutex
	GetAddrs       func(int) []string
	fixtures       map[string]*Runner
	wg             sync.WaitGroup
	fixtureFactory FixtureFactory
}

// NewSupervisor returns a new *Supervisor ready to produce test fixtures for
// your tests using ff. NewSupervisor should be called at most once per package.
// Calline NewSupervisor more than once will split up test summaries and lead to
// less useful output. In future it may panic to prevent this.
func NewSupervisor(ff FixtureFactory) *Supervisor {
	return &Supervisor{
		fixtureFactory: ff,
		fixtures:       map[string]*Runner{},
	}
}

// NewRunner returns a new *Runner ready to run tests with all possible
// combinations of the provided Matrix. NewRunner should be called exactly once
// in each top-level TestXXX(t *testing.T) function in your package. Calling it
// more than once per top-level test may cause undefined behaviour and may
// panic.
func (pfs *Supervisor) NewRunner(t *testing.T, m Matrix) *Runner {
	if Flags.PrintMatrix {
		matrix := m.scenarios()
		for _, m := range matrix {
			fmt.Printf("%s/%s\n", t.Name(), m)
		}
		t.Skip("Just printing test matrix (-ls-matrix flag set)")
	}
	t.Helper()
	t.Parallel()
	pf := &Runner{
		t:                t,
		matrix:           m,
		testNames:        map[string]struct{}{},
		testNamesPassed:  map[string]struct{}{},
		testNamesSkipped: map[string]struct{}{},
		testNamesFailed:  map[string]struct{}{},
		parent:           pfs,
	}
	pfs.mu.Lock()
	defer pfs.mu.Unlock()
	pfs.fixtures[t.Name()] = pf
	return pf
}

// PrintSummary prints a summary of tests run by top-level test and as a sum
// total. It reports tests failed, skipped, passed, and missing (when a test has
// failed to report back any status, which should not happen under normal
// circumstances.
func (pfs *Supervisor) PrintSummary() {
	pfs.wg.Wait()
	pfs.mu.Lock()
	defer pfs.mu.Unlock()
	var total, passed, skipped, failed, missing []string
	for _, pf := range pfs.fixtures {
		t, p, s, f, m := pf.printSummary()
		total = append(total, t...)
		passed = append(passed, p...)
		skipped = append(skipped, s...)
		failed = append(failed, f...)
		missing = append(missing, m...)
	}

	if len(failed) != 0 {
		fmt.Printf("These tests failed:\n")
		for _, n := range failed {
			fmt.Printf("FAILED> %s\n", n)
		}
	}

	if len(missing) != 0 {
		fmt.Printf("These tests did not report status:\n")
		for _, n := range missing {
			fmt.Printf("MISSING> %s\n", n)
		}
	}

	summary := fmt.Sprintf("Summary: %d failed; %d skipped; %d passed; %d missing (total %d)",
		len(failed), len(skipped), len(passed), len(missing), len(total))
	fmt.Fprintln(os.Stdout, summary)
}
