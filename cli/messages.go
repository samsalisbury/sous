package cli

import (
	"fmt"
	"time"

	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
)

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

type cliResultMessage struct {
	res        cmdr.Result
	start, end time.Time
}

func reportCliResult(logsink logging.LogSink, start time.Time, res cmdr.Result) *cliResultMessage {
	msg := &cliResultMessage{
		callerInfo: logging.GetCallerInfo("cli/cli"),
		res:        res,
		start:      start,
		end:        time.Now(),
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
	f("@loglov3-otl", "sous-cli-result-v1")
	msg.callerInfo.EachField(f)
	f("exit-code", msg.res.ExitCode())
	f("duration", msg.end.Sub(msg.start))
}
