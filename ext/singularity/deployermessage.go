package singularity

import (
	"fmt"
	"io"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type deployerMessage struct {
	logging.CallerInfo
	logging.Level
	msg  string
	pair *sous.DeployablePair
}

func reportDeployerMessage(message string, pair *sous.DeployablePair, level logging.Level, logger logging.LogSink) {
	msg := deployerMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		Level:      level,
		msg:        message,
		pair:       pair,
	}
	logging.Deliver(msg, logger)
}

func (msg deployerMessage) DefaultLevel() logging.Level {
	return msg.Level
}

func (msg deployerMessage) Message() string {
	return msg.msg
}

func (msg deployerMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-rectifier-singularity-v1")
	f("pair-id", msg.pair.ID)
	f("request-id", msg.pair.request.ID)
	msg.CallerInfo.EachField(f)
}

func (msg deployerMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", msg.msg)
}
