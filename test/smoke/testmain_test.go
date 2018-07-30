//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"
)

var sup *Supervisor

func TestMain(m *testing.M) {
	flag.BoolVar(&flags.printMatrix, "ls", false, "list test matrix names")
	flag.BoolVar(&flags.printDimensions, "dimensions", false, "list test matrix dimensions")
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
