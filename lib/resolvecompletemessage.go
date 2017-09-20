package sous

import (
	"time"

	"github.com/opentable/sous/util/logging"
)

type resolveCompleteMessage struct {
	logging.CallerInfo
	logging.Level
	status *ResolveStatus
}

func reportResolverStatus(logger logging.LogSink, status *ResolveStatus) {
	msg := resolveCompleteMessage{
		CallerInfo: logging.GetCallerInfo("reportResolverStatus"),
		Level:      logging.InformationLevel,
		status:     status,
	}
	logging.Deliver(msg, logger)
}

func (msg resolveCompleteMessage) MetricsTo(m logging.MetricsSink) {
	if msg.status.Started.Before(msg.status.Finished) {
		m.UpdateTimer("fullcycle-duration", msg.status.Finished.Sub(msg.status.Started))
	}
}

func (msg resolveCompleteMessage) DefaultLevel() logging.Level {
	if !msg.status.Started.Before(msg.status.Finished) {
		return logging.WarningLevel
	}
	if len(msg.status.Errs.Causes) > 0 {
		return logging.WarningLevel
	}
	return logging.InformationLevel
}

func (msg resolveCompleteMessage) Message() string {
	if !msg.status.Started.Before(msg.status.Finished) {
		return "Recording stable status - started time not before finished"
	}
	return "Recording stable status"
}

func (msg resolveCompleteMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-resolution-result-v1")
	f("started-at", msg.status.Started.Format(time.RFC3339))
	f("finished-at", msg.status.Finished.Format(time.RFC3339))
	f("errors", msg.status.Errs.Causes)
	f("resolutions", msg.status.Log)
	f("intended", msg.status.Intended)
	msg.CallerInfo.EachField(f)
}
