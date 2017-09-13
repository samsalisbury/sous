package logging

import (
	"github.com/davecgh/go-spew/spew"
)

type silentMessageError struct {
	CallerInfo
	Level
	message interface{}
}

// ReportClientHTTPResponse reports a response recieved by Sous as a client, as provided as fields.
func reportSilentMessage(logger LogSink, message interface{}) {
	m := newSilentMessageError(message)
	Deliver(m, logger)
}

func newSilentMessageError(message interface{}) *silentMessageError {
	return &silentMessageError{
		Level:      CriticalLevel,
		CallerInfo: GetCallerInfo(),

		message: message,
	}
}

func (msg *silentMessageError) MetricsTo(metrics MetricsSink) {
	metrics.IncCounter("silent-messages", 1)
}

func (msg *silentMessageError) EachField(f FieldReportFn) {
	f("@loglov3-otl", "msg-v1")
	msg.CallerInfo.EachField(f)
}

func (msg *silentMessageError) Message() string {
	return spew.Sprintf("%+#v", msg.message)
}
