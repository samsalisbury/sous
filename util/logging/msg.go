package logging

type genericMsg struct {
	//CallerInfo
	CallTime
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
		CallTime:   GetCallTime(),

		message: msg,
		fields:  fields,
	}
}

func (msg *genericMsg) EachField(f FieldReportFn) {
	if _, hasSchema := msg.fields["@loglov3-otl"]; !hasSchema {
		f("@loglov3-otl", "msg-v1")
	}
	for k, v := range msg.fields {
		f(k, v)
	}
	msg.CallTime.EachField(f)
	msg.CallerInfo.EachField(f)
}

func (msg *genericMsg) Message() string {
	return msg.message
}
