//+build smoke

package smoke

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

var pfs = newParallelTestFixtureSet(PTFOpts{
	NumFreeAddrs: 128,
})

var flags = struct {
	printMatrix bool
}{}

func TestMain(m *testing.M) {
	flag.BoolVar(&flags.printMatrix, "ls-matrix", false, "list test matrix names")
	flag.Parse()
	if flags.printMatrix {
		matrix := fixtureConfigs()
		for _, m := range matrix {
			fmt.Println(m.Desc())
		}
		os.Exit(0)
	}
	resetSingularity()
	exitCode := m.Run()
	pfs.PrintSummary()
	os.Exit(exitCode)
}

func resetSingularity() {
	envDesc := getEnvDesc()
	singularity := NewSingularity(envDesc.SingularityURL())
	singularity.Reset()
}
