package cli

import (
	"fmt"
	"time"

	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
)

type invocationMessage struct {
	callerInfo CallerInfo
	args       []string
	interval   MessageInterval
}

func reportInvocation(ls LogSink, start time.Time, args []string) {
	msg := newInvocationMessage(args, start)
	msg.callerInfo.ExcludeMe()
	Deliver(msg, ls)
}

func newInvocationMessage(args []string, start time.Time) *invocationMessage {
	return &invocationMessage{
		callerInfo: GetCallerInfo(NotHere()),
		args:       args,
		interval:   CompleteInterval(start),
	}
}

func (msg *invocationMessage) DefaultLevel() Level {
	return InformationLevel
}

func (msg *invocationMessage) Message() string {
	return "Invoked"
}

func (msg *invocationMessage) EachField(f FieldReportFn) {
	f("@loglov3-otl", SousCliV1)
	msg.callerInfo.EachField(f)
	msg.interval.EachField(f)
	f("arguments", fmt.Sprintf("%q", msg.args))
}

type cliResultMessage struct {
	callerInfo CallerInfo
	args       []string
	res        cmdr.Result
	interval   MessageInterval
}

func reportCLIResult(logsink LogSink, args []string, start time.Time, res cmdr.Result) {
	msg := newCLIResult(args, start, res)
	msg.callerInfo.ExcludeMe()
	Deliver(msg, logsink)
}

func newCLIResult(args []string, start time.Time, res cmdr.Result) *cliResultMessage {
	return &cliResultMessage{
		callerInfo: GetCallerInfo(NotHere()),
		args:       args,
		res:        res,
		interval:   CompleteInterval(start),
	}
}

func (msg *cliResultMessage) DefaultLevel() Level {
	return InformationLevel
}

func (msg *cliResultMessage) Message() string {
	return fmt.Sprintf("Returned result: %q", msg.res)
}

func (msg *cliResultMessage) EachField(f FieldReportFn) {
	f("@loglov3-otl", SousCliV1)
	msg.callerInfo.EachField(f)
	msg.interval.EachField(f)
	f("arguments", fmt.Sprintf("%q", msg.args))
	f("exit-code", msg.res.ExitCode())
}
