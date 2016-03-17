package cli

import "fmt"

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
	// InternalError signifies programmer error. The user only sees these when
	// we mess up.
	InternalError Error
	// UsageError signifies that the user made a mistake with the invocation.
	UsageError Error
	// OSError signifies that something went wrong starting a process, or
	// performing some other os-level operation.
	OSError Error
	// IOError signifies that something went wrong with io, to files, or accross
	// the network, for example.
	IOError Error
	// UnknownError is the error of last resort, only to be used if none of the
	// other error types is applicable.
	UnknownError Error
)

// EnsureErrorResult takes an error, and if it is not already also a Result,
// makes it an UnknownError. Otherwise, the original error is left unchanged.
func EnsureErrorResult(err error) Result {
	if result, ok := err.(Result); ok {
		return result
	}
	return UnknownError{Err: err}
}

func Errorf(err error, format string, v ...interface{}) Error {
	return Error{Err: err, Message: fmt.Sprintf(format, v...)}
}
func InternalErrorf(err error, format string, v ...interface{}) InternalError {
	return InternalError(Errorf(err, format, v...))
}
func UsageErrorf(err error, format string, v ...interface{}) UsageError {
	return UsageError(Errorf(err, format, v...))
}
func OSErrorf(err error, format string, v ...interface{}) OSError {
	return OSError(Errorf(err, format, v...))
}
func IOErrorf(err error, format string, v ...interface{}) IOError {
	return IOError(Errorf(err, format, v...))
}
func UnknownErrorf(err error, format string, v ...interface{}) UnknownError {
	return UnknownError(Errorf(err, format, v...))
}

func (e InternalError) ExitCode() int { return EX_SOFTWARE }
func (e UsageError) ExitCode() int    { return EX_USAGE }
func (e OSError) ExitCode() int       { return EX_OSERR }
func (e IOError) ExitCode() int       { return EX_IOERR }
func (e UnknownError) ExitCode() int  { return 255 }

func (e InternalError) Error() string { return (Error)(e).Pre("internal") }
func (e UsageError) Error() string    { return (Error)(e).Pre("usage") }
func (e OSError) Error() string       { return (Error)(e).Pre("os") }
func (e IOError) Error() string       { return (Error)(e).Pre("io") }
func (e UnknownError) Error() string  { return (Error)(e).Pre("unknown") }

func (e InternalError) UserTip() string { return e.Tip }
func (e UsageError) UserTip() string    { return e.Tip }
func (e OSError) UserTip() string       { return e.Tip }
func (e IOError) UserTip() string       { return e.Tip }
func (e UnknownError) UserTip() string  { return e.Tip }

func (e Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Err)
}

func (e Error) Pre(prefix string) string {
	return fmt.Sprintf("%s error: %s", prefix, e.Error())
}
