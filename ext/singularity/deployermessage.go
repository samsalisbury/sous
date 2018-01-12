package singularity

import (
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/lib"
	"io"
	"fmt"
)

type deployerMessage struct {
	logging.CallerInfo
	logging.Level
	msg string
	pair *sous.DeployablePair
}

func logDeployerMessage(message string, logger logging.LogSink, pair *sous.DeployablePair) {
	msg := deployerMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		Level: 		logging.InformationLevel,
		msg:		message,
		pair:		pair,
	}
	logging.Deliver(msg, logger)
}

func (msg deployerMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg deployerMessage) Message() string {
	return msg.msg
}

func (msg deployerMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-generic-v1")
	msg.CallerInfo.EachField(f)
}

func (msg deployerMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", msg.msg)
}