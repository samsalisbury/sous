package docker

import "github.com/opentable/sous/util/logging"

var (
	// Log is an alias to logging.Log
	Log = logging.Log
)

func log(ls logging.LogSink, msg string, lvl logging.Level, data ...interface{}) {
	logging.Deliver(ls, append([]interface{}{
		logging.SousGenericV1, logging.MessageField(msg), lvl, logging.GetCallerInfo(logging.NotHere()),
	}, data...)...)
}
