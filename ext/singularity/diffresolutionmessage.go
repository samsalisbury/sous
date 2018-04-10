package singularity

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type diffResolutionMessage struct {
	logging.CallerInfo
	logging.Level
	msg            string
	diffResolution sous.DiffResolution
}

func reportDiffResolutionMessage(message string, diffRes sous.DiffResolution, level logging.Level, logger logging.LogSink) {
	msg := diffResolutionMessage{
		CallerInfo:     logging.GetCallerInfo(logging.NotHere()),
		Level:          level,
		msg:            message,
		diffResolution: diffRes,
	}
	logging.NewDeliver(logger, msg)
}

func (msg diffResolutionMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousDiffResolution)
	f("sous-deployment-id", msg.diffResolution.DeploymentID.String())
	f("sous-manifest-id", msg.diffResolution.ManifestID.String())
	f("sous-resolution-description", string(msg.diffResolution.Desc))
	if msg.diffResolution.Error != nil {
		f("sous-resolution-errormessage", msg.diffResolution.Error.String)
		f("sous-resolution-errortype", msg.diffResolution.Error.Type)
	}
	msg.CallerInfo.EachField(f)
}

func (msg diffResolutionMessage) Message() string {
	return msg.msg
}
