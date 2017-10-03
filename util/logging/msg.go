package logging

type genericMsg struct {
	CallerInfo
	Level
	message string
	fields  map[string]interface{}
}

// ReportMsg is an appropriate for most off-the-cuff logging. It probably calls
// to be replaced with something more specialized, though.
func ReportMsg(logger LogSink, lvl Level, msg string) {
	m := NewGenericMsg(lvl, msg, nil)
	Deliver(m, logger)
}

// NewGenericMsg creates an event out of a map of fields. There are no metrics
// associated with the event - for that you need to define a specialized
// message type.
func NewGenericMsg(lvl Level, msg string, fields map[string]interface{}) LogMessage {
	return &genericMsg{
		Level:      lvl,
		CallerInfo: GetCallerInfo(),

		message: msg,
		fields:  fields,
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
