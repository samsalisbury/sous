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
	msg        string
	submessage *sous.DeployablePairSubmessage
	diffs      sous.Differences
	taskData   *singularityTaskData
	error      error
}

type diffResolutionMessage struct {
	logging.CallerInfo
	logging.Level
	msg            string
	diffResolution *sous.DiffResolution
}

func reportDeployerMessage(message string, pair *sous.DeployablePair, diffs sous.Differences, taskData *singularityTaskData, error error, level logging.Level, logger logging.LogSink) {
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

func reportDiffResolutionMessage(message string, diffRes *sous.DiffResolution, level logging.Level, logger logging.LogSink) {
	msg := diffResolutionMessage{
		CallerInfo:     logging.GetCallerInfo(logging.NotHere()),
		Level:          level,
		msg:            message,
		diffResolution: diffRes,
	}
	logging.Deliver(msg, logger)
}

func (msg deployerMessage) DefaultLevel() logging.Level {
	return msg.Level
}
func (msg diffResolutionMessage) DefaultLevel() logging.Level {
	return msg.Level
}

func (msg deployerMessage) Message() string {
	return msg.msg
}
func (msg diffResolutionMessage) Message() string {
	return msg.msg
}

func (msg deployerMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-rectifier-singularity-v1")
	f("diffs", msg.diffs.String())
	if msg.taskData != nil {
		f("request-id", msg.taskData.requestID)
	}
	if msg.error != nil {
		f("error", msg.error.Error())
	}
	msg.CallerInfo.EachField(f)
	msg.submessage.EachField(f)
}
func (msg diffResolutionMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-rectifier-singularity-v1")
	if msg.diffResolution != nil {
		f("deployment-id", msg.diffResolution.DeploymentID.String())
		f("diffresolution-desc", string(msg.diffResolution.Desc))
		if msg.diffResolution.Error != nil {
			f("error", msg.diffResolution.Error.String)
		}
	}
	msg.CallerInfo.EachField(f)
}

func (msg deployerMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", msg.msg)
}
func (msg diffResolutionMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", msg.msg)
}
