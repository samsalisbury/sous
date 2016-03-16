package cli

import "fmt"

type UsageError struct {
	Message, Tip string
}

func (err UsageError) Error() string {
	return err.Message
}

type InternalError struct {
	Message         string
	UnderlyingError error
}

func InternalErrorf(err error, format string, v ...interface{}) InternalError {
	return InternalError{
		Message:         fmt.Sprintf(format, v...),
		UnderlyingError: err,
	}
}

func (err InternalError) Error() string {
	return fmt.Sprintf("%s: %s", err.Message, err.UnderlyingError)
}
