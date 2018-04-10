package logging

import (
	"fmt"
	"io"
)

type errorMessage struct {
	CallerInfo
	err     error
	console bool
}

// ReportErrorConsole is used to report an error via structured logging.
// If you need more information than "an error occurred", consider using a
// different structured message.
func ReportErrorConsole(sink LogSink, err error) {
	ReportError(sink, err, true)
}

// ReportError is used to report an error via structured logging.
// If you need more information than "an error occurred", consider using a
// different structured message.
func ReportError(sink LogSink, err error, console ...bool) {

	useConsole := false
	if len(console) > 0 {
		useConsole = console[0]
	}

	msg := newErrorMessage(err, useConsole)
	msg.CallerInfo.ExcludeMe()
	NewDeliver(sink, msg)
}

func newErrorMessage(err error, console bool) *errorMessage {
	return &errorMessage{
		CallerInfo: GetCallerInfo(NotHere()),
		err:        err,
		console:    console,
	}
}

func (msg *errorMessage) DefaultLevel() Level {
	return WarningLevel
}

func (msg *errorMessage) Message() string {
	return msg.err.Error()
}

func (msg *errorMessage) WriteToConsole(console io.Writer) {
	if msg.console {
		fmt.Fprintf(console, "%s\n", msg.Message())
	}
}

func (msg *errorMessage) EachField(fn FieldReportFn) {
	fn("@loglov3-otl", SousErrorV1)
	msg.CallerInfo.EachField(fn)
	fn("sous-error-msg", msg.err.Error())
	fn("sous-error-backtrace", fmt.Sprintf("%+v", msg.err))
}
