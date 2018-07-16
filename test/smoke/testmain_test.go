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
	printMatrix bool
}{}

type fixtureConfig struct {
	dbPrimary  bool
	startState *sous.State
	projects   ProjectList
	Desc       string
}

func Matrix() MatrixDef {
	m := NewMatrix()
	m.AddDimension("store", map[string]interface{}{
		"db":  true,
		"git": false,
	})
	m.AddDimension("builder", map[string]interface{}{
		"simple": projects.SingleDockerfile,
		"split":  projects.SplitBuild,
	})
	return m
}

func TestMain(m *testing.M) {
	flag.BoolVar(&flags.printMatrix, "ls-matrix", false, "list test matrix names")
	flag.Parse()
	if flags.printMatrix {

	} else {
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
