package logging

import (
	"fmt"
	"io"
)

type logConfigurationError struct {
	CallerInfo
	message string
}

func reportLogConfigurationError(ls LogSet, msg string) {
	warning := newLogConfigurationError(msg)
	warning.ExcludeMe()
	NewDeliver(ls, warning)
}

func newLogConfigurationError(msg string) *logConfigurationError {
	return &logConfigurationError{
		message:    msg,
		CallerInfo: GetCallerInfo(NotHere()),
	}
}

func (l *logConfigurationError) DefaultLevel() Level {
	return WarningLevel
}

func (l *logConfigurationError) Message() string {
	return l.message
}

func (l *logConfigurationError) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "problem configuring logging: %s", l.message)
}

func (l *logConfigurationError) Error() string {
	return ConsoleError(l)
}
