package sous

import (
	"fmt"

	"github.com/opentable/sous/util/logging"
)

// droppedR11nMessage is a specialisation of diffRezMessage, so we embed that
// anonymously and override Message.
type droppedR11nMessage struct {
	*diffRezMessage
	reason string
}

func (msg *droppedR11nMessage) Message() string {
	return fmt.Sprintf("rectification dropped: %s", msg.reason)
}

func reportDroppedR11n(ls logging.LogSink, r *Rectification, reason string) {
	msg := &droppedR11nMessage{
		diffRezMessage: &diffRezMessage{
			callerInfo: logging.GetCallerInfo(logging.NotHere()),
			resolution: &DiffResolution{
				DeploymentID: r.Pair.ID(),
				Desc:         r.Pair.Kind().ExpectedResolutionType() + " (not attempted)",
			},
		},
		reason: reason,
	}
	logging.Deliver(msg, ls)
}
