package messages

import (
	"fmt"
	"io"

	"github.com/opentable/sous/util/logging"
)

type (
	logFieldsMessage struct {
		logging.CallerInfo
		logging.Level
		msg             string
		console         bool
		serverConsole   bool
		withIDs         bool
		items           []interface{}
		eachFielders    []logging.EachFielder
		consoleMessages []logging.ConsoleMessage
		metricsMessages []logging.MetricsMessage
	}
)

//ReportLogFieldsMessageWithIDs report message with Ids
func ReportLogFieldsMessageWithIDs(msg string, loglvl logging.Level, logSink logging.LogSink, items ...interface{}) {
	logMessage := buildLogFieldsMessage(msg, false, true, loglvl, items...)
	logMessage.CallerInfo.ExcludeMe()
	logging.NewDeliver(logSink, logMessage)

}

//ReportLogFieldsMessageToConsole report message to console
func ReportLogFieldsMessageToConsole(msg string, loglvl logging.Level, logSink logging.LogSink, items ...interface{}) {
	logMessage := buildLogFieldsMessage(msg, true, false, loglvl, items...)
	logMessage.CallerInfo.ExcludeMe()
	logging.NewDeliver(logSink, logMessage)
}

//ReportLogFieldsMessage generate a logFieldsMessage log entry
func ReportLogFieldsMessage(msg string, loglvl logging.Level, logSink logging.LogSink, items ...interface{}) {
	logMessage := buildLogFieldsMessage(msg, false, true, loglvl, items...)
	logMessage.CallerInfo.ExcludeMe()
	logging.NewDeliver(logSink, logMessage)
}

func buildLogFieldsMessage(msg string, console bool, withIDs bool, loglvl logging.Level, items ...interface{}) logFieldsMessage {
	logMessage := logFieldsMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		Level:      loglvl,
		msg:        msg,
		console:    console,
		withIDs:    withIDs,
		items:      items,
	}

	return logMessage

}

func (l logFieldsMessage) WriteToConsole(console io.Writer) {
	if l.console {
		fmt.Fprintf(console, "%s\n", l.composeMsg())
	}
}

func (l logFieldsMessage) composeMsg() string {
	return l.msg
}

//DefaultLevel return the default log level for this message
func (l logFieldsMessage) DefaultLevel() logging.Level {
	return l.Level
}

//Message return the message string associate with message
func (l logFieldsMessage) Message() string {
	return l.composeMsg()
}

//EachField will make sure individual fields are added for OTL
func (l logFieldsMessage) EachField(fn logging.FieldReportFn) {
	(&l).partitionItems()
	strayfields := assembleStrayFields(l.withIDs, l.items...)

	fn.All(append(l.eachFielders, l.CallerInfo, logging.SousGenericV1, strayfields)...)
}

// holding item partitioning to logging time.
func (l *logFieldsMessage) partitionItems() {
	if l.eachFielders != nil {
		return
	}
	others := []interface{}{}
	for i := 0; i < len(l.items); i++ {
		ef, isef := l.items[i].(logging.EachFielder)
		cm, iscm := l.items[i].(logging.ConsoleMessage)
		mm, ismm := l.items[i].(logging.MetricsMessage)
		if isef {
			l.eachFielders = append(l.eachFielders, ef)
		}
		if iscm {
			l.consoleMessages = append(l.consoleMessages, cm)
		}
		if ismm {
			l.metricsMessages = append(l.metricsMessages, mm)
		}
		if !(isef || iscm || ismm) {
			others = append(others, l.items[i])
		}
	}
	l.items = others
}
