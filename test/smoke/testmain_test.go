//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
)

var pfs = newParallelTestFixtureSet(PTFOpts{
	NumFreeAddrs: 128,
})

var flags = struct {
	printMatrix     bool
	printDimensions bool
}{}

type fixtureConfig struct {
	dbPrimary  bool
	startState *sous.State
	projects   ProjectList
	Desc       string
}

func Matrix() MatrixDef {
	m := NewMatrix()
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
	if flags.printMatrix {

	}

	if runRealTests {
		resetSingularity()
	}
	exitCode := m.Run()
	pfs.PrintSummary()
	os.Exit(exitCode)
}

func resetSingularity() {
	envDesc := getEnvDesc()
	singularity := NewSingularity(envDesc.SingularityURL())
	singularity.Reset()
}
