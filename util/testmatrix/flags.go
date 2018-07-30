package testmatrix

import (
	"flag"
)

// Flags are extra go test flags you can pass to testmatrix.
var Flags = struct {
	PrintMatrix     bool
	PrintDimensions bool
}{}

func init() {
	flag.BoolVar(&Flags.PrintDimensions, "dimensions", false, "list test matrix dimensions")
	flag.BoolVar(&Flags.PrintMatrix, "ls", false, "list test matrix names")
}

// Init must be called from TestMain after flag.Parse, to initialise a new
// Supervisor. If Init returns nil, then tests will not be run this time (e.g.
// because we are just listing tests or printing the matrix def etc.)
func Init(defaultMatrix func() Matrix, f FixtureFactory) *Supervisor {

	runRealTests := !(Flags.PrintMatrix || Flags.PrintDimensions)

	if Flags.PrintDimensions {
		defaultMatrix().PrintDimensions()
	}

	if runRealTests {
		return NewSupervisor(f)
	}
	return nil
}
