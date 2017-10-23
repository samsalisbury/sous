package actions

import (
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type (
	updateBeginMessage struct {
		tries int
		sid   sous.SourceID
		did   sous.DeploymentID
		user  sous.User
		start time.Time
	}

	updateSuccessMessage struct {
		tries int
		sid   sous.SourceID
		did   sous.DeploymentID
		user  sous.User
		start time.Time
	}

	updateErrorMessage struct {
		tries int
		sid   sous.SourceID
		did   sous.DeploymentID
		user  sous.User
		start time.Time
		err   error
	}
)

func newUpdateBeginMessage(tries int, sid sous.SourceID, did sous.DeploymentID, user sous.User, start time.Time) updateBeginMessage {
	return updateBeginMessage{
		callerInfo: logging.GetCallerInfo("cli/actions"),
		tries:      tries,
		sid:        sid,
		did:        did,
		user:       user,
		start:      start,
	}
}

func newUpdateSuccessMessage(tries int, sid sous.SourceID, did sous.DeploymentID, user sous.User, start time.Time) updateSuccessMessage {
	return updateSuccessMessage{
		callerInfo: logging.GetCallerInfo("cli/actions"),
		tries:      tries,
		sid:        sid,
		did:        did,
		user:       user,
		start:      start,
	}
}

func newUpdateErrorMessage(tries int, sid sous.SourceID, did sous.DeploymentID, user sous.User, start time.Time, err error) updateErrorMessage {
	return updateErrorMessage{
		callerInfo: logging.GetCallerInfo("cli/actions"),
		tries:      tries,
		sid:        sid,
		did:        did,
		user:       user,
		start:      start,
		err:        err,
	}
}
