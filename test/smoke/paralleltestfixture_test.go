//+build smoke

package smoke

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

// PTFOpts are options for a ParallelTestFixture.
type PTFOpts struct {
	// NumFreeAddrs determines how many free addresses are guaranteed by this
	// test fixture.
	NumFreeAddrs int
}

type ParallelTestFixture struct {
	NextAddr          func() string
	testNames         map[string]struct{}
	testNamesMu       sync.Mutex
	testNamesPassed   map[string]struct{}
	testNamesPassedMu sync.Mutex
}

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
		NextAddr:        nextAddr,
		testNames:       map[string]struct{}{},
		testNamesPassed: map[string]struct{}{},
	}
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

func (pf *ParallelTestFixture) recordTestPassed(t *testing.T) {
	t.Helper()
	name := t.Name()
	pf.testNamesPassedMu.Lock()
	defer pf.testNamesPassedMu.Unlock()
	if _, ok := pf.testNames[name]; !ok {
		t.Errorf("test reported as passed but not started!?")
	}
	pf.testNamesPassed[name] = struct{}{}
}

func (pf *ParallelTestFixture) PrintSummary(t *testing.T) {
	t.Logf("Test summary: %d out of %d tests passed", len(pf.testNames), len(pf.testNamesPassed))
	fmt.Fprintf(os.Stdout, "Test summary: %d out of %d tests passed", len(pf.testNames), len(pf.testNamesPassed))
}

func (pf *ParallelTestFixture) NewIsolatedFixture(t *testing.T) TestFixture {
	t.Helper()
	pf.recordTestStarted(t)
	return newTestFixture(t, pf, pf.NextAddr)
}
