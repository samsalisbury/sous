package testmatrix

// Init must be called from TestMain after flag.Parse, to initialise a new
// Supervisor. If Init returns nil, then tests will not be run this time (e.g.
// because we are just listing tests or printing the matrix def etc.)
func Init(defaultMatrix func() Matrix, f FixtureFactory, beforeAll func() error) *Supervisor {
	runRealTests := !(Flags.PrintMatrix || Flags.PrintDimensions)
	if Flags.PrintDimensions {
		defaultMatrix().PrintDimensions()
	}
	if runRealTests {
		if err := beforeAll(); err != nil {
			panic(err)
		}
		return NewSupervisor(f)
	}
	return nil
}
