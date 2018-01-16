package logging

import (
	"fmt"
	"io"
)

type genericMsg struct {
	CallerInfo
	Level
	message string
	fields  map[string]interface{}
	console bool
}

// ReportMsg is an appropriate for most off-the-cuff logging. It probably calls
// to be replaced with something more specialized, though.
func ReportMsg(logger LogSink, lvl Level, msg string, console ...bool) {
	useConsole := false
	if len(console) > 0 {
		useConsole = console[0]
	}
	m := NewGenericMsg(lvl, msg, nil, console)
	m.ExcludeMe()
	Deliver(m, logger)
}

func (msg *genericMsg) WriteToConsole(console io.Writer) {
	if msg.console {
		fmt.Fprintf(console, "%s\n", msg)
	}
}

// NewGenericMsg creates an event out of a map of fields. There are no metrics
// associated with the event - for that you need to define a specialized
// message type.
func NewGenericMsg(lvl Level, msg string, fields map[string]interface{}, console bool) *genericMsg {
	return &genericMsg{
		Level:      lvl,
		CallerInfo: GetCallerInfo(NotHere()),

		message: msg,
		fields:  fields,
		console: console,
	}
}

func (msg *genericMsg) EachField(f FieldReportFn) {
	// XXX belongs maybe in the top level structured message engine
	if _, hasSchema := msg.fields["@loglov3-otl"]; !hasSchema {
		f("@loglov3-otl", "sous-generic-v1")
	}
	for k, v := range msg.fields {
		f(k, v)
	}
	msg.CallerInfo.EachField(f)
}

func (msg *genericMsg) Message() string {
	return msg.message
}
