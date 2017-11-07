package sous

import (
	"fmt"
	"io"

	"github.com/opentable/sous/util/logging"
)

type (
	pollerStartMessage struct {
		callerInfo logging.CallerInfo
		poller     *StatusPoller
	}

	subreportMessage struct {
		callerInfo logging.CallerInfo
		poller     *StatusPoller
		update     pollResult
	}

	pollerStatusMessage struct {
		callerInfo    logging.CallerInfo
		poller        *StatusPoller
		previousState ResolveState
	}

	pollerResolvedMessage struct {
		callerInfo logging.CallerInfo
		poller     *StatusPoller
		status     ResolveState
		err        error
	}
)

func reportPollerStart(logsink logging.LogSink, poller *StatusPoller) {
	msg := newPollerStartMessage(poller)
	msg.callerInfo.ExcludeMe()
	logging.Deliver(msg, logsink)
}

func reportPollerResolved(logsink logging.LogSink, sp *StatusPoller, status ResolveState, err error) {
	msg := newPollerResolvedMessage(sp, status, err)
	msg.callerInfo.ExcludeMe()
	logging.Deliver(msg, logsink)
}

func reportPollerStatus(logsink logging.LogSink, poller *StatusPoller, old ResolveState) {
	msg := newPollerStatusMessage(poller, old)
	msg.callerInfo.ExcludeMe()
	logging.Deliver(msg, logsink)
}

func reportSubreport(logsink logging.LogSink, poller *StatusPoller, update pollResult) {
	msg := newSubreportMessage(poller, update)
	msg.callerInfo.ExcludeMe()
	logging.Deliver(msg, logsink)
}

func resultFields(f logging.FieldReportFn, update pollResult) {
	f("update-url", update.url)
	f("update-status", update.stat.String())
	f("update-resolve-id", update.resolveID)
	if update.err != nil {
		f("error", update.err.Error())
	}
}

func pollerFields(f logging.FieldReportFn, sp *StatusPoller) {
	if sp == nil {
		return
	}
	resolveFilterFields(f, sp.ResolveFilter)
	userFields(f, sp.User)
}

func userFields(f logging.FieldReportFn, user User) {
	f("user-name", user.Name)
	f("user-email", user.Email)
}

func resolveFilterFields(f logging.FieldReportFn, rf *ResolveFilter) {
	if rf == nil {
		return
	}
	f("filter-cluster", rf.Cluster.ValueOr("*"))
	f("filter-repo", rf.Repo.ValueOr("*"))
	f("filter-offset", rf.Offset.ValueOr("*"))
	f("filter-tag", rf.Tag.ValueOr("*"))
	f("filter-revision", rf.Revision.ValueOr("*"))
	f("filter-flavor", rf.Flavor.ValueOr("*"))
}

func newPollerStartMessage(poller *StatusPoller) *pollerStartMessage {
	return &pollerStartMessage{
		callerInfo: logging.GetCallerInfo(logging.NotHere()),
		poller:     poller,
	}
}

func (msg *pollerStartMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *pollerStartMessage) Message() string {
	return "Deployment polling starting"
}

func (msg *pollerStartMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-status-polling-v1")
	msg.callerInfo.EachField(f)
	pollerFields(f, msg.poller)
}

func newPollerResolvedMessage(sp *StatusPoller, status ResolveState, err error) *pollerResolvedMessage {
	return &pollerResolvedMessage{
		callerInfo: logging.GetCallerInfo(logging.NotHere()),
		poller:     sp,
		status:     status,
		err:        err,
	}
}

func (msg *pollerResolvedMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *pollerResolvedMessage) Message() string {
	return "Status polling complete"
}

func (msg *pollerResolvedMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-status-polling-v1")
	msg.callerInfo.EachField(f)
	pollerFields(f, msg.poller)
	f("deploy-status", msg.status.String())
}

func (msg *pollerResolvedMessage) WriteToConsole(console io.Writer) {
	if msg.poller == nil {
		return
	}
	rf := msg.poller.ResolveFilter
	if rf == nil {
		return
	}
	sourceLocation := rf.Repo.ValueOr("")
	if rf.Offset.ValueOr("") != "" {
		sourceLocation += ("," + rf.Offset.ValueOr(""))
	}
	if rf.Flavor.ValueOr("") != "" {
		sourceLocation += ("~" + rf.Flavor.ValueOr(""))
	}
	version := rf.Tag.ValueOr("")
	clusterName := rf.Cluster.ValueOr("")
	fmt.Fprintf(console, "%s %s successfully deployed to %s.\n", sourceLocation, version, clusterName)
}

func newPollerStatusMessage(poller *StatusPoller, old ResolveState) *pollerStatusMessage {
	return &pollerStatusMessage{
		callerInfo:    logging.GetCallerInfo(logging.NotHere()),
		poller:        poller,
		previousState: old,
	}
}

func (msg *pollerStatusMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

func (msg *pollerStatusMessage) Message() string {
	return "updated status"
}

func (msg *pollerStatusMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-status-polling-v1")
	msg.callerInfo.EachField(f)
	pollerFields(f, msg.poller)
	f("deploy-status", msg.poller.status.String())
}

func (msg *pollerStatusMessage) WriteToConsole(console io.Writer) {
	if msg.poller.status == msg.previousState {
		console.Write([]byte{'.'})
		return
	}
	fmt.Fprintf(console, "\n%s", msg.poller.status.Prose())
	if msg.poller.finished() {
		console.Write([]byte{'\n'})
	}
}

//	reportSubreport(sp.logs, sp, update)

func newSubreportMessage(poller *StatusPoller, update pollResult) *subreportMessage {
	return &subreportMessage{
		callerInfo: logging.GetCallerInfo(logging.NotHere()),
		poller:     poller,
		update:     update,
	}
}

func (msg *subreportMessage) DefaultLevel() logging.Level {
	return logging.DebugLevel
}

func (msg *subreportMessage) Message() string {
	return "poll result received from cluster"
}

func (msg *subreportMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-polling-subresult-v1")
	msg.callerInfo.EachField(f)
	pollerFields(f, msg.poller)
	resultFields(f, msg.update)
	if state, exists := msg.poller.statePerCluster[msg.update.url]; exists {
		if state.LastCycle {
			f("resolve-cycle-status", "server resolution includes client's update")
		} else {
			f("resolve-cycle-status", "server resolution not yet guaranteed to include our update")
		}
	}
}
