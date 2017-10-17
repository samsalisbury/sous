package sous

import "github.com/opentable/sous/util/logging"

type pollerStartMessage struct {
	callerInfo logging.CallerInfo
	poller     *StatusPoller
}

func reportPollerStart(logsink logging.LogSink, poller *StatusPoller) {
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

func (msg *pollerStartMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-status-polling-v1")
	msg.callerInfo.EachField(f)
	// ResolveFilter
	// User - maybe for every command?
}
