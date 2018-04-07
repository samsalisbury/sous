package singularity

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type deployerMessage struct {
	logging.CallerInfo
	logging.Level
	msg        string
	submessage logging.Submessage
	diffs      sous.Differences
	taskData   *singularityTaskData
	error      error
}

func reportDeployerMessage(
	message string,
	pair *sous.DeployablePair,
	diffs sous.Differences,
	taskData *singularityTaskData,
	error error,
	level logging.Level,
	logger logging.LogSink,
) {
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

func (msg deployerMessage) Message() string {
	return msg.msg
}

func (msg deployerMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousRectifierSingularityV1)
	f("sous-diffs", msg.diffs.String())
	if msg.taskData != nil {
		f("sous-request-id", msg.taskData.requestID)
	}
	if msg.error != nil {
		f("error", msg.error.Error())
	}
	msg.CallerInfo.EachField(f)
	msg.submessage.EachField(f)
}
