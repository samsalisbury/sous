package logging

import "fmt"

type errorMessage struct {
	CallerInfo
	err error
}

// ReportError is used to report an error via structured logging.
// If you need more information than "an error occurred", consider using a
// different structured message.
func ReportError(sink LogSink, err error) {
	msg := newErrorMessage(err)
	msg.CallerInfo.ExcludeMe()
	Deliver(msg, sink)
}

func newErrorMessage(err error) *errorMessage {
	return &errorMessage{
		CallerInfo: GetCallerInfo(NotHere()),
		err:        err,
	}
}

func (msg *errorMessage) DefaultLevel() Level {
	return WarningLevel
}

func (msg *errorMessage) Message() string {
	return msg.err.Error()
}

func (msg *errorMessage) EachField(fn FieldReportFn) {
	fn("@loglov3-otl", "sous-error-v1")
	msg.CallerInfo.EachField(fn)
	fn("sous-error-msg", msg.err.Error())
	fn("sous-error-backtrace", fmt.Sprintf("%+v", msg.err))
}
