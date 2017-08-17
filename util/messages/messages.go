// The goal of this package is to integrate structured logging an metrics
// reporting with error handling in an interface as close as possible to the
// fluency of fmt.Errorf(...)
package messages

type (
	metricser interface{}
	logger    interface{}
	// error interface{}
)

// New(name string, ...args) error
