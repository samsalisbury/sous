//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"
)

var pfs = newParallelTestFixtureSet(PTFOpts{
	NumFreeAddrs: 128,
})

func TestMain(m *testing.M) {
	flag.Parse()
	exitCode := m.Run()
	pfs.PrintSummary()
	os.Exit(exitCode)
}
