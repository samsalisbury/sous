package smoke

import (
	"testing"

	"github.com/opentable/sous/util/testmatrix"
)

// sup is the global matrix supervisor, used to collate all test results.
var sup *testmatrix.Supervisor

// runner is a wrapper around Runner allowing fully baked fixtures to be
// passed directly to tests, rather than the test having to unwrap scenarios
// themselves.
type runner struct{ *testmatrix.Runner }

// Run is analogous to Runner.Run, but accepts a func in terms of strongly typed
// fixture rather than having to manually unwrap scenarios.
func (r *runner) Run(name string, test func(*testing.T, *fixture)) {
	r.Runner.Run(name, func(t *testing.T, c testmatrix.Context) {
		test(t, c.F.(*fixture))
	})
}

type scenarioTest func(*testing.T, testmatrix.Scenario, *testmatrix.LateFixture)

type fixtureConfigFunc func(*fixtureConfig)

func (r *runner) RunScenario(name string, ff fixtureConfigFunc, test testmatrix.Test) {
	r.Runner.RunScenario(name, func(t *testing.T, s testmatrix.Scenario, lf *testmatrix.LateFixture) {
		scenario := unwrapScenario(s)
		fix := newConfiguredFixture(t, scenario, ff)
		lf.Set(fix)
	})
}

// newRunner should be called once at the start of every top-level package
// test to produce that test's matrixRunner.
func newRunner(t *testing.T, m testmatrix.Matrix) runner {
	return runner{Runner: sup.NewRunner(t, m)}
}
