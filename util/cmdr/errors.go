package cmdr

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

type (
	// An ErrorResult is both an error and a result, and has a tip for the user.
	ErrorResult interface {
		error
		Result
		WithTip(string) ErrorResult
		WithUnderlyingError(error) ErrorResult
		Tipper
	}
	// Error is a generic error, and should only be used when none of the other
	// error types are applicable. Note that it implements error but not Result,
	// so it cannot be used by itself to return from commands. This is by
	// design, use one of the specialised error types below, which all implement
	// Result.
	cliErr struct {
		// Message is the main message to tell the user what went wrong.
		Message,
		// Tip is a tip to the user, to help them avoid this error in future.
		Tip string
		// Err is an underlying error, if any, which may also be shown to the
		// user.
		Err error
	}
	// InternalErr signifies programmer error. The user only sees these when
	// we mess up.
	InternalErr struct{ *cliErr }
	// UsageErr signifies that the user made a mistake with the invocation.
	UsageErr struct{ *cliErr }
	// OSErr signifies that something went wrong starting a process, or
	// performing some other os-level operation.
	OSErr struct{ *cliErr }
	// IOErr signifies that something went wrong with io, to files, or across
	// the network, for example.
	IOErr struct{ *cliErr }
	// UnknownErr is the error of last resort, only to be used if none of the
	// other error types is applicable.
	UnknownErr struct{ *cliErr }
)

// EnsureErrorResult takes an error, and if it is not already also a Result,
// makes it into an ErrorResult. It tries to wrap well-known errors
// intelligently, and eventially falls back to UnknownErr if no sensible
// ErrorResult exists for that error.
func EnsureErrorResult(err error) ErrorResult {
	err = errors.Cause(err)
	if result, ok := err.(ErrorResult); ok {
		return result
	}
	if pathErr, ok := err.(*os.PathError); ok {
		return OSErr{&cliErr{Err: pathErr}}
	}
	return UnknownErr{&cliErr{Err: err}}
}

func newError(format string, v ...interface{}) *cliErr {
	return &cliErr{Message: fmt.Sprintf(format, v...)}
}

func InternalErrorf(format string, v ...interface{}) InternalErr {
	return InternalErr{newError(format, v...)}
}

func UsageErrorf(format string, v ...interface{}) UsageErr {
	return UsageErr{newError(format, v...)}
}

func OSErrorf(format string, v ...interface{}) OSErr {
	return OSErr{newError(format, v...)}
}

func IOErrorf(format string, v ...interface{}) IOErr {
	return IOErr{newError(format, v...)}
}

func UnknownErrorf(format string, v ...interface{}) UnknownErr {
	return UnknownErr{newError(format, v...)}
}

func (e InternalErr) ExitCode() int { return EX_SOFTWARE }
func (e UsageErr) ExitCode() int    { return EX_USAGE }
func (e OSErr) ExitCode() int       { return EX_OSERR }
func (e IOErr) ExitCode() int       { return EX_IOERR }
func (e UnknownErr) ExitCode() int  { return 255 }
func (e *cliErr) ExitCode() int     { return 255 }

func (e *cliErr) UserTip() string { return e.Tip }

func (e *cliErr) Error() string {
	if e.Err == nil {
		return e.Message
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Err)
	}
	return e.Err.Error()
}

func (e *cliErr) WithTip(tip string) ErrorResult {
	e.Tip = tip
	return e
}

func (e *cliErr) WithUnderlyingError(err error) ErrorResult {
	e.Err = err
	return e
}

func (e *cliErr) prefix(prefix string) string {
	return fmt.Sprintf("%s error: %s", prefix, e.Error())
}
