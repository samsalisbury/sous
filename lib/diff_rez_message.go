package sous

import "github.com/opentable/sous/util/logging"

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
	f("@loglov3-otl", "sous-diff-resolution")
	msg.callerInfo.EachField(f)
	f("sous-deployment-id", msg.resolution.DeploymentID.String())
	f("sous-manifest-id", msg.resolution.ManifestID.String())
	f("sous-diff-source-type", "global rectifier")
	f("sous-diff-source-user", "unknown")
	f("sous-resolution-description", string(msg.resolution.Desc))
	if msg.resolution.Error != nil {
		marshallable := buildMarshableError(msg.resolution.Error.error)
		f("sous-resolution-errortype", marshallable.Type)
		f("sous-resolution-errormessage", marshallable.String)
	} else {
		f("sous-resolution-errortype", "")
		f("sous-resolution-errormessage", "")
	}
}
