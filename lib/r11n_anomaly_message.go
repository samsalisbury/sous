package sous

import (
	"fmt"

	"github.com/opentable/sous/util/logging"
)

// r11nAnomalyMessage is a specialisation of diffRezMessage, so we embed that
// anonymously and override Message and DefaultLevel.
type r11nAnomalyMessage struct {
	*diffRezMessage
	anomaly r11nAnomaly
}

type r11nAnomaly int

const (
	// r11nWentMissing means the rectification was accepted but went missing
	// for reasons unknown before reporting a result.
	r11nWentMissing r11nAnomaly = iota
	// r11nDroppedQueueNotEmpty means the r11n was never attempted because the
	// queue for that DeploymentID was not empty.
	r11nDroppedQueueNotEmpty
	// R11nDroppedQueueFull means the r11n qwas never attempted because the
	// queue for that DeploymentID was full.
	r11nDroppedQueueFull // not yet used but needed very soon.
)

func (a r11nAnomaly) note() string {
	switch a {
	default:
		return "unidentified anomaly"
	case r11nDroppedQueueNotEmpty, r11nDroppedQueueFull:
		return "not attempted"
	case r11nWentMissing:
		return "went missing"
	}
}

func (a r11nAnomaly) action() string {
	switch a {
	default:
		return "unidentified anomaly"
	case r11nDroppedQueueNotEmpty, r11nDroppedQueueFull:
		return "dropped"
	case r11nWentMissing:
		return "went missing"
	}
}

func (a r11nAnomaly) reason() string {
	switch a {
	default:
		return "reason unknown"
	case r11nDroppedQueueNotEmpty:
		return "queue not empty"
	case r11nDroppedQueueFull:
		return "queue full"
	}
}

func (msg *r11nAnomalyMessage) Message() string {
	return fmt.Sprintf("rectification %s: %s",
		msg.anomaly.action(), msg.anomaly.reason())
}

func newR11nAnomalyMessage(r *Rectification, anomaly r11nAnomaly) *r11nAnomalyMessage {
	desc := fmt.Sprintf("not %s (%s)",
		r.Pair.Kind().ExpectedResolutionType(), anomaly.note())
	return &r11nAnomalyMessage{
		diffRezMessage: &diffRezMessage{
			callerInfo: logging.GetCallerInfo(logging.NotHere()),
			resolution: &DiffResolution{
				DeploymentID: r.Pair.ID(),
				// Note in some places, consts of ResolutionType are treated
				// like an enum, the ResolutionType in this log message will not
				// be recognised by such code.
				Desc: ResolutionType(desc),
			},
		},
		anomaly: anomaly,
	}
}

func reportR11nAnomaly(ls logging.LogSink, r *Rectification, anomaly r11nAnomaly) {
	logging.Deliver(newR11nAnomalyMessage(r, anomaly), ls)
}
