package cli

import (
	"fmt"

	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
)

type cliResultMessage struct {
	res cmdr.Result
}

func reportCliResult(logsink logging.LogSink, res cmdr.Result) *cliResultMessage {
	msg := &cliResultMessage{
		callerInfo: logging.GetCallerInfo("cli/cli"),
		res:        res,
	}
	logging.Deliver(msg, logsink)
}

func (msg *cliResultMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *cliResultMessage) Message() string {
	return fmt.Sprintf("Returned result: %q", msg.res)
}

func (msg *cliResultMessage) EachField(f FieldReportFn) {
	f("@loglov3-otl", "sous-generic-v1")
	msg.callerInfo.EachField(f)
}

type invocationMessage struct {
	args []string
}

func reportInvocation(logsink logging.LogSink, args []string) *invocationMessage {
	msg := &invocationMessage{
		callerInfo: logging.GetCallerInfo("cli/cli"),
		args:       args,
	}
	logging.Deliver(msg, logsink)
}

func (msg *invocationMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *invocationMessage) Message() string {
	return fmt.Sprintf("Invoked with: %q", msg.args)
}

func (msg *invocationMessage) EachField(f FieldReportFn) {
	f("@loglov3-otl", "sous-generic-v1")
	msg.callerInfo.EachField(f)
}

func reportInvocation(args []string, ls logging.LogSink) {
	msg := invocationMessage{
		callerInfo: logging.GetCallerInfo("cli/cli"),
		args:       args,
	}
	logging.Deliver(msg, ls)
}
