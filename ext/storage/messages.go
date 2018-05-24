package storage

import (
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type (
	storeMessage struct {
		logging.CallerInfo
		logging.MessageInterval
		direction direction
		state     *sous.State
		err       error
	}

	direction uint
)

const (
	read direction = iota
	write
)

func reportReading(log logging.LogSink, started time.Time, state *sous.State, err error) {
	msg := newStoreMessage(started, read, state, err)
	msg.CallerInfo.ExcludeMe()
	logging.Deliver(log, msg)
}

func reportWriting(log logging.LogSink, started time.Time, state *sous.State, err error) {
	msg := newStoreMessage(started, write, state, err)
	msg.CallerInfo.ExcludeMe()
	logging.Deliver(log, msg)
}

func newStoreMessage(started time.Time, dir direction, state *sous.State, err error) *storeMessage {
	return &storeMessage{
		CallerInfo:      logging.GetCallerInfo(logging.NotHere()),
		MessageInterval: logging.NewInterval(started, time.Now()),
		state:           state,
		direction:       dir,
		err:             err,
	}
}

// DefaultLevel implements LogMessage on storeMessage.
func (msg *storeMessage) DefaultLevel() logging.Level {
	if msg.err == nil {
		return logging.DebugLevel
	}
	return logging.WarningLevel
}

// Message implements LogMessage on storeMessage.
func (msg *storeMessage) Message() string {
	return msg.direction.message()
}

func (dir direction) message() string {
	switch dir {
	default:
		return "Unknown state storage direction (shouldn't ever occur?)"
	case read:
		return "Reading state"
	case write:
		return "Writing state"
	}
}

func (dir direction) String() string {
	if dir == write {
		return "write"
	}
	return "read"
}

// EachField implements LogMessage on storeMessage.
func (msg *storeMessage) EachField(fn logging.FieldReportFn) {
	fn("@loglov3-otl", logging.SousGenericV1)
	msg.CallerInfo.EachField(fn)
	msg.MessageInterval.EachField(fn)
	if msg.err != nil {
		fn("sous-storage-error", msg.err.Error())
	}
	deps, err := msg.state.Deployments()
	if err == nil {
		fn("sous-storage-deployments", deps.Len())
	} else {
		fn("sous-storage-deployments", 0)
	}
}
