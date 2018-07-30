//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"
)

// sup is the global matrix supervisor, used to collate all test results.
var sup *Supervisor

// matrixRunner is a wrapper around Runner allowing fully baked fixtures to be
// passed directly to tests, rather than the test having to unwrap scenarios
// themselves.
type matrixRunner struct{ r *Runner }

// Run is analogous to Runner.Run, but accepts a func in terms of strongly typed
// fixture rather than having to manually unwrap scenarios.
func (m *matrixRunner) Run(name string, test func(*testing.T, *testFixture)) {
	m.r.Run(name, func(t *testing.T, c Context) { test(t, c.F.(*testFixture)) })
}

// newMatrixRunner should be called once at the start of every top-level package
// test to produce that test's matrixRunner.
func newMatrixRunner(t *testing.T, m Matrix) matrixRunner {
	return matrixRunner{r: sup.NewRunner(t, m)}
}

func TestMain(m *testing.M) {
	flag.BoolVar(&flags.printDimensions, "dimensions", false, "list test matrix dimensions")
	flag.BoolVar(&flags.printMatrix, "ls", false, "list test matrix names")
	flag.Parse()

	runRealTests := !(flags.printMatrix || flags.printDimensions)

	if flags.printDimensions {
		matrix().PrintDimensions()
	}

	if runRealTests {
		sup = NewSupervisor(newTestFixture)
		resetSingularity()
	}
	exitCode := m.Run()
	sup.PrintSummary()
	os.Exit(exitCode)
}

func resetSingularity() {
	envDesc := getEnvDesc()
	singularity := newSingularity(envDesc.SingularityURL())
	singularity.Reset()
}
