package sous

import (
	"github.com/opentable/sous/util/logging"
)

type deployableMessage struct {
	pairmessage logging.Submessage
	callerInfo  logging.CallerInfo
}

func (msg *deployableMessage) DefaultLevel() logging.Level {
	return msg.pairmessage.RecommendedLevel()
}

func (msg *deployableMessage) Message() string {
	//return msg.pairmessage.pair.Kind().String() + " deployment diff"
	return "deployment diff"
}

func (msg *deployableMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousDeploymentDiff)
	msg.callerInfo.EachField(f)
	msg.pairmessage.EachField(f)
}
