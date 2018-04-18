package singularity

// XXX deprecated - remove in favor of bare Delivers
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
	logging.Deliver(logger, msg)
}

func (msg diffResolutionMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousDiffResolution)
	msg.CallerInfo.EachField(f)

	msg.diffResolution.EachField(f)
}

func (msg diffResolutionMessage) Message() string {
	return msg.msg
}
