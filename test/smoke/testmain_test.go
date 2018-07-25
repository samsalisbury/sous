//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"
)

var pfs *parallelTestFixtureSet

// Matrix returns the defined sous smoke test matrix.
func Matrix() matrixDef {
	m := newMatrix()
	m.AddDimension("store", "GDM storage to use", map[string]interface{}{
		"db":  true,
		"git": false,
	})
	m.AddDimension("project", "type of project to build", map[string]interface{}{
		"simple": projects.SingleDockerfile,
		"split":  projects.SplitBuild,
	})
	return m
}

func TestMain(m *testing.M) {
	flag.BoolVar(&flags.printMatrix, "ls", false, "list test matrix names")
	flag.BoolVar(&flags.printDimensions, "dimensions", false, "list test matrix dimensions")
	flag.Parse()

	runRealTests := !(flags.printMatrix || flags.printDimensions)

	if flags.printDimensions {
		Matrix().PrintDimensions()
	}

	if runRealTests {
		pfs = newParallelTestFixtureSet()
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
