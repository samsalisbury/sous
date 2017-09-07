package logging

import (
	"fmt"
	"io"
)

type logConfigurationError struct {
	CallerInfo
	CallTime
	message string
}

func reportLogConfigurationError(ls LogSet, msg string) {
	warning := newLogConfigurationError(msg)
	Deliver(warning, ls)
}

func newLogConfigurationError(msg string) *logConfigurationError {
	return &logConfigurationError{
		message:    msg,
		CallTime:   GetCallTime(),
		CallerInfo: GetCallerInfo("reportLogConfigurationWarning", "newLogConfigurationWarning"),
	}
}

func (l *logConfigurationError) DefaultLevel() Level {
	return WarningLevel
}

func (l *logConfigurationError) Message() string {
	return l.message
}

func (l *logConfigurationError) EachField(f FieldReportFn) {
	l.CallTime.EachField(f)
	l.CallerInfo.EachField(f)
}

func (l *logConfigurationError) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "problem configuring logging: %s", l.message)
}

func (l *logConfigurationError) Error() string {
	return ConsoleError(l)
}
