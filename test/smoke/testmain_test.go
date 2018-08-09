//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"

	"github.com/opentable/sous/util/testmatrix"
)

func TestMain(m *testing.M) {
	flag.Parse()
	testmatrix.Quiet = quiet()
	sup = testmatrix.Init(matrix, newFixture, func() error {
		resetSingularity()
		return stopPIDs()
	})
	exitCode := m.Run()
	sup.PrintSummary()
	os.Exit(exitCode)
}

func resetSingularity() {
	envDesc := getEnvDesc()
	singularity := newSingularity(envDesc.SingularityURL())
	singularity.Reset()
}
