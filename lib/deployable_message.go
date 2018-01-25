package sous

import (
	"github.com/opentable/sous/util/logging"
)

type deployableMessage struct {
	submessage *DeployablePairSubmessage
	callerInfo logging.CallerInfo
}

func (msg *deployableMessage) DefaultLevel() logging.Level {
	if msg.submessage.pair.Post == nil {
		return logging.WarningLevel
	}

	if msg.submessage.pair.Prior == nil {
		return logging.InformationLevel
	}

	if len(msg.submessage.pair.Diffs()) == 0 {
		return logging.DebugLevel
	}

	return logging.InformationLevel
}

func (msg *deployableMessage) Message() string {
	return msg.submessage.pair.Kind().String() + " deployment diff"
}

func (msg *deployableMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-deployment-diff")
	msg.callerInfo.EachField(f)
	msg.submessage.EachField(f)
}
