package logging

type genericMsg struct {
	//CallerInfo
	CallTime
	Level
	message string
}

// ReportClientHTTPResponse reports a response recieved by Sous as a client, as provided as fields.
func ReportMsg(logger LogSink, lvl Level, msg string) {
	m := newGenericMsg(lvl, msg)
	Deliver(m, logger)
}

func newGenericMsg(lvl Level, msg string) *genericMsg {
	return &genericMsg{
		Level: lvl,
		//CallerInfo: GetCallerInfo(),
		CallTime: GetCallTime(),

		message: msg,
	}
}

func (msg *genericMsg) EachField(f FieldReportFn) {
	f("@loglov3-otl", "msg-v1")
	msg.CallTime.EachField(f)
}

func (msg *genericMsg) Message() string {
	return msg.message
}
