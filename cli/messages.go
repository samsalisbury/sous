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
	interval   logging.MessageInterval
}

func reportInvocation(ls logging.LogSink, start time.Time, args []string) {
	msg := newInvocationMessage(args, start)
	msg.callerInfo.ExcludeMe()
	logging.Deliver(ls, msg)
}

func newInvocationMessage(args []string, start time.Time) *invocationMessage {
	return &invocationMessage{
		callerInfo: logging.GetCallerInfo(logging.NotHere()),
		args:       args,
		interval:   logging.CompleteInterval(start),
	}
}

func (msg *invocationMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *invocationMessage) Message() string {
	return "Invoked"
}

func (msg *invocationMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousCliV1)
	msg.callerInfo.EachField(f)
	msg.interval.EachField(f)
	f("arguments", fmt.Sprintf("%q", msg.args))
}

type cliResultMessage struct {
	callerInfo logging.CallerInfo
	args       []string
	res        cmdr.Result
	interval   logging.MessageInterval
}

func reportCLIResult(logsink logging.LogSink, args []string, start time.Time, res cmdr.Result) {
	msg := newCLIResult(args, start, res)
	msg.callerInfo.ExcludeMe()
	logging.Deliver(logsink, msg)
}

func newCLIResult(args []string, start time.Time, res cmdr.Result) *cliResultMessage {
	return &cliResultMessage{
		callerInfo: logging.GetCallerInfo(logging.NotHere()),
		args:       args,
		res:        res,
		interval:   logging.CompleteInterval(start),
	}
}

func (msg *cliResultMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *cliResultMessage) Message() string {
	return fmt.Sprintf("Returned result: %q", msg.res)
}

func (msg *cliResultMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousCliV1)
	msg.callerInfo.EachField(f)
	msg.interval.EachField(f)
	f("arguments", fmt.Sprintf("%q", msg.args))
	f("exit-code", msg.res.ExitCode())
}
