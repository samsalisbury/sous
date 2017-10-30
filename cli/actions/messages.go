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
		tries      int
		sid        sous.SourceID
		did        sous.DeploymentID
		user       sous.User
		interval   logging.MessageInterval
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
		interval:   logging.OpenInterval(start),
	}
}

func newUpdateSuccessMessage(tries int, sid sous.SourceID, did sous.DeploymentID, user sous.User, start time.Time) updateMessage {
	return updateMessage{
		callerInfo: logging.GetCallerInfo("cli/actions"),
		tries:      tries,
		sid:        sid,
		did:        did,
		user:       user,
		interval:   logging.CompleteInterval(start),
	}
}

func newUpdateErrorMessage(tries int, sid sous.SourceID, did sous.DeploymentID, user sous.User, start time.Time, err error) updateMessage {
	return updateMessage{
		callerInfo: logging.GetCallerInfo("cli/actions"),
		tries:      tries,
		sid:        sid,
		did:        did,
		user:       user,
		interval:   logging.OpenInterval(start),
		err:        err,
	}
}

func (msg updateMessage) Message() string {
	if msg.err != nil {
		return "Error during update"
	}
	if msg.interval.Complete() {
		return "Update successful"
	}
	return "Beginning update"
}

func (msg updateMessage) EachField(fn logging.FieldReportFn) {
	fn("@loglov3-otl", "sous-update-v1")
	msg.callerInfo.EachField(fn)
	msg.interval.EachField(fn)

	fn("try-number", msg.tries)
	fn("source-id", msg.sid.String())
	fn("deploy-id", msg.did.String())
	fn("user-email", msg.user.Email)

	if msg.err != nil {
		fn("error", msg.err.Error())
	}
}

func (msg updateMessage) WriteToConsole(console io.Writer) {
	if msg.err != nil {
		console.Write([]byte(msg.err.Error()))
		return
	}
	if msg.interval.Complete() {
		console.Write([]byte("Updated global manifest"))
	}
}
