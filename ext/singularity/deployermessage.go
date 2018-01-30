package singularity

import (
	"fmt"
	"io"
	"strings"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type deployerMessage struct {
	logging.CallerInfo
	logging.Level
	msg        string
	submessage *sous.DeployablePairSubmessage
	diffs      *sous.Differences
	taskData   *singularityTaskData
	error      error
}

func reportDeployerMessage(message string, pair *sous.DeployablePair, diffs *sous.Differences, taskData *singularityTaskData, error error, level logging.Level, logger logging.LogSink) {
	msg := deployerMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		Level:      level,
		msg:        message,
		submessage: sous.NewDeployablePairSubmessage(pair),
		diffs:      diffs,
		taskData:   taskData,
		error:      error,
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
	f("diffs", strings.Join(*msg.diffs, "\n"))
	f("request-id", msg.taskData.requestID)
	f("error", msg.error.Error())
	msg.CallerInfo.EachField(f)
	msg.submessage.EachField(f)
}

func (msg deployerMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", msg.msg)
}
