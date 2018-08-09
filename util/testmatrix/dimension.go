package testmatrix

// Dimension represents a dimension in the test matrix.
type Dimension struct {
	// Name is the name of this dimension, used to look up values in calculated
	// Scenarios.
	Name string
	// Desc is used in help text when using the -matrix flag.
	Desc string
	// Values is a map of named possible values for this Dimension.
	// The name used here forms part of the sub-test path.
	Values map[string]interface{}
}
