package temporary

import "fmt"

// IsTemporary returns true when the passed error is temporary.
func IsTemporary(err error) bool {
	t, ok := err.(interface {
		Temporary() bool
	})
	return ok && t.Temporary()
}

// Errorf returns an error implementing Temporary() bool, which always returns
// true.
func Errorf(format string, a ...interface{}) error {
	return tempErr{error: fmt.Errorf(format, a...)}
}

// WrapError makes returns an error implementing Temporary() bool, which always
// returns true.
func WrapError(err error) error {
	return tempErr{error: err}
}

type tempErr struct {
	error
}

// Temporary always returns true.
func (te tempErr) Temporary() bool { return true }
