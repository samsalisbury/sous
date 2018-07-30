package smoke

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

type (
	// Runner runs tests defined in a Matrix.
	Runner struct {
		t                  *testing.T
		matrix             Matrix
		testNames          map[string]struct{}
		testNamesMu        sync.RWMutex
		testNamesPassed    map[string]struct{}
		testNamesPassedMu  sync.Mutex
		testNamesSkipped   map[string]struct{}
		testNamesSkippedMu sync.Mutex
		testNamesFailed    map[string]struct{}
		testNamesFailedMu  sync.Mutex
		parent             *Supervisor
	}

	// Supervisor supervises a set of Runners, and collates their results.
	// There should be exactly one global Supervisor in every package that uses
	// testmatrix.
	Supervisor struct {
		mu             sync.Mutex
		GetAddrs       func(int) []string
		fixtures       map[string]*Runner
		wg             sync.WaitGroup
		fixtureFactory FixtureFactory
	}
)

// NewSupervisor returns a new *Supervisor ready to produce test fixtures for
// your tests using ff. NewSupervisor should be called at most once per package.
// Calline NewSupervisor more than once will split up test summaries and lead to
// less useful output. In future it may panic to prevent this.
func NewSupervisor(ff FixtureFactory) *Supervisor {
	if err := stopPIDs(); err != nil {
		panic(err)
	}
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
	if flags.printMatrix {
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

func (pf *Runner) recordTestStarted(t *testing.T) {
	t.Helper()
	name := t.Name()
	pf.testNamesMu.Lock()
	defer pf.testNamesMu.Unlock()
	if _, ok := pf.testNames[name]; ok {
		t.Fatalf("duplicate test name: %q", name)
	}
	pf.testNames[name] = struct{}{}
}

// Test is a test.
type Test func(*testing.T, Context)

// FixtureFactory generates Fixtures from test and combination.
type FixtureFactory func(*testing.T, Scenario) Fixture

// Fixture is able to set up and tear down.
type Fixture interface {
	Teardown(*testing.T)
}

// Context is passed to each test case.
type Context struct {
	Scenario Scenario
	// F is the fixture returned from FixtureFactory().F()
	F interface{}
}

// Run is analogous to *testing.T.Run, but takes a method that includes a
// Context as well as *testing.T. Run runs the defined test with all possible
// matrix combinations in parallel.
func (pf *Runner) Run(name string, test Test) {
	for _, c := range pf.matrix.scenarios() {
		c := c
		pf.t.Run(c.String()+"/"+name, func(t *testing.T) {
			pf.parent.wg.Add(1)
			f := pf.parent.fixtureFactory(t, c)
			defer func() {
				defer pf.parent.wg.Done()
				pf.recordTestStatus(t)
				f.Teardown(t)
			}()
			pf.recordTestStarted(t)
			test(t, Context{Scenario: c, F: f})
		})
	}
}

func (pf *Runner) recordTestStatus(t *testing.T) {
	t.Helper()
	name := t.Name()
	pf.testNamesMu.RLock()
	_, started := pf.testNames[name]
	pf.testNamesMu.RUnlock()

	statusString := "UNKNOWN"
	status := &statusString
	defer func() { rtLog("Finished running %s: %s", name, *status) }()

	if !started {
		t.Fatalf("test %q reported as finished, but not started", name)
		*status = "ERROR: Not Started"
		return
	}
	switch {
	default:
		*status = "PASSED"
		pf.testNamesPassedMu.Lock()
		pf.testNamesPassed[name] = struct{}{}
		pf.testNamesPassedMu.Unlock()
		return
	case t.Skipped():
		*status = "SKIPPED"
		pf.testNamesSkippedMu.Lock()
		pf.testNamesSkipped[name] = struct{}{}
		pf.testNamesSkippedMu.Unlock()
		return
	case t.Failed():
		*status = "FAILED"
		pf.testNamesFailedMu.Lock()
		pf.testNamesFailed[name] = struct{}{}
		pf.testNamesFailedMu.Unlock()
		return
	}
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

func testNamesSlice(m map[string]struct{}) []string {
	var s, i = make([]string, len(m)), 0
	for n := range m {
		s[i] = n
		i++
	}
	return s
}

func (pf *Runner) printSummary() (total, passed, skipped, failed, missing []string) {
	t := pf.t
	t.Helper()
	total = testNamesSlice(pf.testNames)
	passed = testNamesSlice(pf.testNamesPassed)
	skipped = testNamesSlice(pf.testNamesSkipped)
	failed = testNamesSlice(pf.testNamesFailed)

	if !quiet() {
		summary := fmt.Sprintf("%s summary: %d failed; %d skipped; %d passed (total %d)",
			t.Name(), len(failed), len(skipped), len(passed), len(total))
		t.Log(summary)
		fmt.Fprintln(os.Stdout, summary)
	}

	missingCount := len(total) - (len(passed) + len(failed) + len(skipped))
	if missingCount != 0 {
		for t := range pf.testNamesPassed {
			delete(pf.testNames, t)
		}
		for t := range pf.testNamesSkipped {
			delete(pf.testNames, t)
		}
		for t := range pf.testNamesFailed {
			delete(pf.testNames, t)
		}
		for t := range pf.testNames {
			missing = append(missing, t)
		}
	}
	return total, passed, skipped, failed, missing
}
