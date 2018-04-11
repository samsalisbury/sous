package logging

type kafkaSendErrorMessage struct {
	callerInfo CallerInfo
	err        error
}

func reportKafkaSendError(logsink LogSink, err error) {
	msg := newKafkaSendErrorMessage(err)
	msg.callerInfo.ExcludeMe()
	Deliver(logsink, msg)
}

func newKafkaSendErrorMessage(err error) *kafkaSendErrorMessage {
	return &kafkaSendErrorMessage{
		callerInfo: GetCallerInfo(NotHere()),
		err:        err,
	}
}

func (msg *kafkaSendErrorMessage) DefaultLevel() Level {
	return WarningLevel
}

func (msg *kafkaSendErrorMessage) Message() string {
	return "Error sending message to kafka"
}

func (msg *kafkaSendErrorMessage) EachField(f FieldReportFn) {
	f("@loglov3-otl", SousGenericV1)
	msg.callerInfo.EachField(f)
	f("error", msg.err.Error())
}
