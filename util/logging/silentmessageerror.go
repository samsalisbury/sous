package logging

import (
	"fmt"
	"io"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/util/logging/constants"
)

type silentMessageError struct {
	CallerInfo
	Level
	message interface{}
}

func reportSilentMessage(logger LogSink, message interface{}) {
	m := newSilentMessageError(message)
	m.ExcludeMe()
	Deliver(m, logger)
}

func newSilentMessageError(message interface{}) *silentMessageError {
	return &silentMessageError{
		Level:      CriticalLevel,
		CallerInfo: GetCallerInfo(NotHere()),

		message: message,
	}
}

func (msg *silentMessageError) MetricsTo(metrics MetricsSink) {
	metrics.IncCounter("silent-messages", 1)
}

func (msg *silentMessageError) EachField(f FieldReportFn) {
	f("@loglov3-otl", constants.SousGenericV1)
	msg.CallerInfo.EachField(f)
}

func (msg *silentMessageError) Message() string {
	return spew.Sprintf("SILENT: %+#v", msg.message)
}

func (msg *silentMessageError) WriteToConsole(w io.Writer) {
	fmt.Fprintf(w, "ERROR: a message of type %T was generated, but it did not emit any output. This is a bug in Sous.\n", msg.message)
}
