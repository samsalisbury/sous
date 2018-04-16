package sous

import (
	"github.com/opentable/sous/util/logging"
)

// XXX deprecated - remote in favor of bare deliver
type diffRezMessage struct {
	resolution *DiffResolution
	callerInfo logging.CallerInfo
}

func (msg diffRezMessage) DefaultLevel() logging.Level {
	return logging.WarningLevel
}

func (msg diffRezMessage) Message() string {
	return string(msg.resolution.Desc)
}

func (msg diffRezMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousDiffResolution)
	msg.callerInfo.EachField(f)
	msg.resolution.EachField(f)
}
