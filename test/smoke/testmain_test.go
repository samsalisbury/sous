//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"

	"github.com/opentable/sous/util/testmatrix"
)

// sup is the global matrix supervisor, used to collate all test results.
var sup *testmatrix.Supervisor

// matrixRunner is a wrapper around Runner allowing fully baked fixtures to be
// passed directly to tests, rather than the test having to unwrap scenarios
// themselves.
type matrixRunner struct{ r *testmatrix.Runner }

// Run is analogous to Runner.Run, but accepts a func in terms of strongly typed
// fixture rather than having to manually unwrap scenarios.
func (m *matrixRunner) Run(name string, test func(*testing.T, *testFixture)) {
	m.r.Run(name, func(t *testing.T, c testmatrix.Context) { test(t, c.F.(*testFixture)) })
}

// newMatrixRunner should be called once at the start of every top-level package
// test to produce that test's matrixRunner.
func newMatrixRunner(t *testing.T, m testmatrix.Matrix) matrixRunner {
	return matrixRunner{r: sup.NewRunner(t, m)}
}

func TestMain(m *testing.M) {
	flag.Parse()
	testmatrix.Quiet = quiet()
	if sup = testmatrix.Init(matrix, newTestFixture); sup != nil {
		resetSingularity()
		if err := stopPIDs(); err != nil {
			panic(err)
		}
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
