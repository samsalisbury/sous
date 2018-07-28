//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"
)

var pfs *parallelTestFixtureSet

func TestMain(m *testing.M) {
	flag.BoolVar(&flags.printMatrix, "ls", false, "list test matrix names")
	flag.BoolVar(&flags.printDimensions, "dimensions", false, "list test matrix dimensions")
	flag.Parse()

	runRealTests := !(flags.printMatrix || flags.printDimensions)

	if flags.printDimensions {
		matrix().PrintDimensions()
	}

	if runRealTests {
		pfs = newParallelTestFixtureSet(newTestFixture)
		resetSingularity()
	}
	exitCode := m.Run()
	pfs.PrintSummary()
	os.Exit(exitCode)
}

func resetSingularity() {
	envDesc := getEnvDesc()
	singularity := newSingularity(envDesc.SingularityURL())
	singularity.Reset()
}
