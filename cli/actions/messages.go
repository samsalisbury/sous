package actions

import (
	"io"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type (
	updateMessage struct {
		callerInfo logging.CallerInfo
		finished   bool
		tries      int
		sid        sous.SourceID
		did        sous.DeploymentID
		user       sous.User
		start      time.Time
		err        error
	}
)

func newUpdateBeginMessage(tries int, sid sous.SourceID, did sous.DeploymentID, user sous.User, start time.Time) updateMessage {
	return updateMessage{
		callerInfo: logging.GetCallerInfo("cli/actions"),
		tries:      tries,
		sid:        sid,
		did:        did,
		user:       user,
		start:      start,
	}
}

func newUpdateSuccessMessage(tries int, sid sous.SourceID, did sous.DeploymentID, user sous.User, start time.Time) updateMessage {
	return updateMessage{
		callerInfo: logging.GetCallerInfo("cli/actions"),
		finished:   true,
		tries:      tries,
		sid:        sid,
		did:        did,
		user:       user,
		start:      start,
	}
}

func newUpdateErrorMessage(tries int, sid sous.SourceID, did sous.DeploymentID, user sous.User, start time.Time, err error) updateMessage {
	return updateMessage{
		callerInfo: logging.GetCallerInfo("cli/actions"),
		tries:      tries,
		sid:        sid,
		did:        did,
		user:       user,
		start:      start,
		err:        err,
	}
}

func (msg updateMessage) Message() string {
	if msg.err != nil {
		return "Error during update"
	}
	if msg.finished {
		return "Update successful"
	}
	return "Beginning update"
}

func (msg updateMessage) EachField(fn logging.FieldReportFn) {
	fn("@loglov3-otl", "sous-update-v1")
	msg.callerInfo.EachField(fn)
	fn("try-number", msg.tries)
	fn("source-id", msg.sid)
	fn("deploy-id", msg.did)
	fn("user-email", msg.user.Email)
	fn("started-at", msg.start)
	if msg.finished {
		fn("finished-at", time.Now())
	}
	if msg.err != nil {
		fn("error", msg.err.Error())
	}
}

func (msg updateMessage) WriteToConsole(console io.Writer) {
	if msg.err != nil {
		console.Write([]byte(msg.err.Error()))
	}
}
