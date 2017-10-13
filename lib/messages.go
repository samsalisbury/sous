package sous

type pollerStartMessage struct {
	poller *StatusPoller
}

func reportPollerStart(logsink logging.LogSink, poller *StatusPoller) *pollerStartMessage {
	msg := &pollerStartMessage{
		callerInfo: logging.GetCallerInfo(),
		poller:     poller,
	}
	logging.Deliver(msg, logsink)
}

func (msg *pollerStartMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *pollerStartMessage) Message() string {
	return "Deployment polling starting"
}

func (msg *pollerStartMessage) EachField(f FieldReportFn) {
	f("@loglov3-otl", "sous-status-polling-v1")
	msg.callerInfo.EachField(f)
	// ResolveFilter
	// User - maybe for every command?
}
