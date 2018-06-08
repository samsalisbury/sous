//+build smoke

package smoke

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	sous "github.com/opentable/sous/lib"
)

type (
	// PTFOpts are options for a ParallelTestFixture.
	PTFOpts struct {
		// NumFreeAddrs determines how many free addresses are guaranteed by this
		// test fixture.
		NumFreeAddrs int
	}

	fixtureConfig struct {
		dbPrimary  bool
		startState *sous.State
	}

	ParallelTestFixture struct {
		NextAddr           func() string
		testNames          map[string]struct{}
		testNamesMu        sync.RWMutex
		testNamesPassed    map[string]struct{}
		testNamesPassedMu  sync.Mutex
		testNamesSkipped   map[string]struct{}
		testNamesSkippedMu sync.Mutex
		testNamesFailed    map[string]struct{}
		testNamesFailedMu  sync.Mutex
	}
)

func resetSingularity(t *testing.T) {
	envDesc := getEnvDesc(t)
	singularity := NewSingularity(envDesc.SingularityURL())
	singularity.Reset(t)
}

func newParallelTestFixture(t *testing.T, opts PTFOpts) *ParallelTestFixture {
	t.Helper()
	resetSingularity(t)
	stopPIDs(t)
	numFreeAddrs := opts.NumFreeAddrs
	freeAddrs := freePortAddrs(t, "127.0.0.1", numFreeAddrs, 6601, 9000)
	var nextAddrIndex int64
	nextAddr := func() string {
		i := atomic.AddInt64(&nextAddrIndex, 1)
		if i == int64(numFreeAddrs) {
			panic("ran out of free ports; increase numFreeAddrs")
		}
		return freeAddrs[i]
	}
	return &ParallelTestFixture{
		NextAddr:         nextAddr,
		testNames:        map[string]struct{}{},
		testNamesPassed:  map[string]struct{}{},
		testNamesSkipped: map[string]struct{}{},
		testNamesFailed:  map[string]struct{}{},
	}
}

func (fcfg fixtureConfig) Desc() string {
	if fcfg.dbPrimary {
		return "DB"
	}
	return "GIT"
}

func (pf *ParallelTestFixture) recordTestStarted(t *testing.T) {
	t.Helper()
	name := t.Name()
	pf.testNamesMu.Lock()
	defer pf.testNamesMu.Unlock()
	if _, ok := pf.testNames[name]; ok {
		t.Fatalf("duplicate test name: %q", name)
	}
	pf.testNames[name] = struct{}{}
}

func (pf *ParallelTestFixture) recordTestStatus(t *testing.T) {
	t.Helper()
	name := t.Name()
	pf.testNamesMu.RLock()
	_, started := pf.testNames[name]
	pf.testNamesMu.RUnlock()
	if !started {
		t.Fatalf("test %q reported as finished, but not started", name)
		return
	}
	switch {
	default:
		pf.testNamesPassedMu.Lock()
		pf.testNamesPassed[name] = struct{}{}
		pf.testNamesPassedMu.Unlock()
		return
	case t.Skipped():
		pf.testNamesSkippedMu.Lock()
		pf.testNamesSkipped[name] = struct{}{}
		pf.testNamesSkippedMu.Unlock()
		return
	case t.Failed():
		pf.testNamesFailedMu.Lock()
		pf.testNamesFailed[name] = struct{}{}
		pf.testNamesFailedMu.Unlock()
		return
	}
}

func (pf *ParallelTestFixture) PrintSummary(t *testing.T) {
	total := len(pf.testNames)
	passed := len(pf.testNamesPassed)
	skipped := len(pf.testNamesSkipped)
	failed := len(pf.testNamesFailed)

	summary := fmt.Sprintf("Test summary: %d failed; %d skipped; %d passed (total %d)", failed, skipped, passed, total)
	t.Log(summary)
	fmt.Fprint(os.Stdout, summary)

	missing := total - (passed + failed + skipped)
	if missing != 0 {
		for t := range pf.testNamesPassed {
			delete(pf.testNames, t)
		}
		for t := range pf.testNamesSkipped {
			delete(pf.testNames, t)
		}
		for t := range pf.testNamesFailed {
			delete(pf.testNames, t)
		}
		var missingTests []string
		for t := range pf.testNames {
			missingTests = append(missingTests, t)
		}
		t.Fatalf("Some tests did not report status: %s", strings.Join(missingTests, ", "))
	}
}

func (pf *ParallelTestFixture) NewIsolatedFixture(t *testing.T, fcfg fixtureConfig) TestFixture {
	t.Helper()
	pf.recordTestStarted(t)
	return newTestFixture(t, pf, pf.NextAddr, fcfg)
}
