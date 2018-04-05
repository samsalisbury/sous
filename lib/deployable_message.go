package sous

import (
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/constants"
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
	f("@loglov3-otl", constants.SousDeploymentDiff)
	msg.callerInfo.EachField(f)
	msg.pairmessage.EachField(f)
}
