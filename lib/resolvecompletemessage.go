package sous

import (
	"time"

	"github.com/opentable/sous/util/logging"
)

type resolveCompleteMessage struct {
	logging.CallerInfo
	logging.Level
	status *ResolveStatus
	logging.MessageInterval
}

func reportResolverStatus(logger logging.LogSink, status *ResolveStatus) {
	msg := resolveCompleteMessage{
		CallerInfo:      logging.GetCallerInfo("reportResolverStatus"),
		Level:           logging.InformationLevel,
		MessageInterval: logging.NewInterval(status.Started, status.Finished),
	}
	logging.Deliver(msg, logger)
}

func (msg resolveCompleteMessage) MetricsTo(m logging.MetricsSink) {
	if msg.status.Started.Before(msg.status.Finished) {
		m.UpdateTimer("fullcycle-duration", msg.status.Finished.Sub(msg.status.Started))
	}
	m.UpdateSample("resolution-errors", int64(len(msg.status.Errs.Causes)))
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
	f("error-count", len(msg.status.Errs.Causes))
	msg.CallerInfo.EachField(f)
	msg.MessageInterval.EachField(f)
}
