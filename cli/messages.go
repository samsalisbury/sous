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

func reportInvocation(args []string, ls logging.LogSink) {
	msg := invocationMessage{
		callerInfo: logging.GetCallerInfo("cli/cli"),
		args:       args,
	}
	logging.Deliver(msg, ls)
}

func (msg *invocationMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *invocationMessage) Message() string {
	return fmt.Sprintf("Invoked with: %q", msg.args)
}

func (msg *invocationMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-generic-v1")
	msg.callerInfo.EachField(f)
}

type cliResultMessage struct {
	callerInfo logging.CallerInfo
	res        cmdr.Result
	start, end time.Time
}

func reportCLIResult(logsink logging.LogSink, start time.Time, res cmdr.Result) {
	msg := newCLIResult(start, res)
	logging.Deliver(msg, logsink)
}

func newCLIResult(start time.Time, res cmdr.Result) *cliResultMessage {
	return &cliResultMessage{
		callerInfo: logging.GetCallerInfo("cli/cli"),
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
	f("@loglov3-otl", "sous-cli-result-v1")
	msg.callerInfo.EachField(f)
	f("exit-code", msg.res.ExitCode())
	f("duration", msg.end.Sub(msg.start))
}
