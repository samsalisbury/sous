package cli

import (
	"fmt"
	"time"

	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
)

type invocationMessage struct {
	callerInfo logging.CallerInfo
	args       []string
}

func reportInvocation(ls logging.LogSink, args []string) {
	logging.Deliver(newInvocationMessage(args), ls)
}

func newInvocationMessage(args []string) *invocationMessage {
	return &invocationMessage{
		callerInfo: logging.GetCallerInfo("cli/messages", "cli/cli"),
		args:       args,
	}
}

func (msg *invocationMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *invocationMessage) Message() string {
	return "Invoked"
}

func (msg *invocationMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-cli-v1")
	msg.callerInfo.EachField(f)
	f("arguments", fmt.Sprintf("%q", msg.args))
}

type cliResultMessage struct {
	callerInfo logging.CallerInfo
	args       []string
	res        cmdr.Result
	start, end time.Time
}

func reportCLIResult(logsink logging.LogSink, args []string, start time.Time, res cmdr.Result) {
	msg := newCLIResult(args, start, res)
	logging.Deliver(msg, logsink)
}

func newCLIResult(args []string, start time.Time, res cmdr.Result) *cliResultMessage {
	return &cliResultMessage{
		callerInfo: logging.GetCallerInfo("cli/messages", "cli/cli"),
		args:       args,
		res:        res,
		start:      start,
		end:        time.Now(),
	}
}

func (msg *cliResultMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *cliResultMessage) Message() string {
	return fmt.Sprintf("Returned result: %q", msg.res)
}

func (msg *cliResultMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-cli-v1")
	msg.callerInfo.EachField(f)
	f("exit-code", msg.res.ExitCode())
	f("duration", msg.end.Sub(msg.start))
	f("arguments", fmt.Sprintf("%q", msg.args))
}
