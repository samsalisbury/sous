package cmdr

import (
	"fmt"
	"os"
)

type (
	// An ErrorResult is both an error and a result, and has a tip for the user.
	ErrorResult interface {
		error
		Result
		Tipper
	}
	// Error is a generic error, and should only be used when none of the other
	// error types are applicable. Note that it implements error but not Result,
	// so it cannot be used by itself to return from commands. This is by
	// design, use one of the specialised error types below, which all implement
	// Result.
	Error struct {
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
	InternalErr Error
	// UsageErr signifies that the user made a mistake with the invocation.
	UsageErr Error
	// OSErr signifies that something went wrong starting a process, or
	// performing some other os-level operation.
	OSErr Error
	// IOErr signifies that something went wrong with io, to files, or across
	// the network, for example.
	IOErr Error
	// UnknownErr is the error of last resort, only to be used if none of the
	// other error types is applicable.
	UnknownErr Error
)

// EnsureErrorResult takes an error, and if it is not already also a Result,
// makes it into an ErrorResult. It tries to wrap well-known errors
// intelligently, and eventially falls back to UnknownErr if no sensible
// ErrorResult exists for that error.
func EnsureErrorResult(err error) ErrorResult {
	if result, ok := err.(ErrorResult); ok {
		return result
	}
	if pathErr, ok := err.(*os.PathError); ok {
		return OSErr{Err: pathErr}
	}
	return UnknownErr{Err: err}
}

func newError(err error, format string, v ...interface{}) Error {
	return Error{Err: err, Message: fmt.Sprintf(format, v...)}
}

func InternalError(err error, format string, v ...interface{}) InternalErr {
	return InternalErr(newError(err, format, v...))
}
func InternalErrorf(format string, v ...interface{}) InternalErr {
	return InternalError(nil, format, v...)
}

func UsageError(err error, format string, v ...interface{}) UsageErr {
	return UsageErr(newError(err, format, v...))
}
func UsageErrorf(format string, v ...interface{}) UsageErr {
	return UsageError(nil, format, v...)
}

func OSError(err error, format string, v ...interface{}) OSErr {
	return OSErr(newError(err, format, v...))
}
func OSErrorf(format string, v ...interface{}) OSErr {
	return OSError(nil, format, v...)
}

func IOError(err error, format string, v ...interface{}) IOErr {
	return IOErr(newError(err, format, v...))
}
func IOErrorf(format string, v ...interface{}) IOErr {
	return IOError(nil, format, v...)
}

func UnknownError(err error, format string, v ...interface{}) UnknownErr {
	return UnknownErr(newError(err, format, v...))
}
func UnknownErrorf(format string, v ...interface{}) UnknownErr {
	return UnknownError(nil, format, v...)
}

func (e InternalErr) ExitCode() int { return EX_SOFTWARE }
func (e UsageErr) ExitCode() int    { return EX_USAGE }
func (e OSErr) ExitCode() int       { return EX_OSERR }
func (e IOErr) ExitCode() int       { return EX_IOERR }
func (e UnknownErr) ExitCode() int  { return 255 }

func (e InternalErr) Error() string { return (Error)(e).prefix("internal") }
func (e UsageErr) Error() string    { return (Error)(e).Error() }
func (e OSErr) Error() string       { return (Error)(e).prefix("os") }
func (e IOErr) Error() string       { return (Error)(e).prefix("io") }
func (e UnknownErr) Error() string  { return (Error)(e).prefix("unknown") }

func (e InternalErr) UserTip() string { return e.Tip }
func (e UsageErr) UserTip() string    { return e.Tip }
func (e OSErr) UserTip() string       { return e.Tip }
func (e IOErr) UserTip() string       { return e.Tip }
func (e UnknownErr) UserTip() string  { return e.Tip }

func (e Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Err)
}

func (e Error) prefix(prefix string) string {
	return fmt.Sprintf("%s error: %s", prefix, e.Error())
}
