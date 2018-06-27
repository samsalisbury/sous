package sous

import (
	"context"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
)

type (
	// StatusPoller polls servers for status.
	StatusPoller struct {
		restful.HTTPClient
		*ResolveFilter
		User            User
		statePerCluster map[string]*pollerState
		status          ResolveState
		logs            logging.LogSink
		results         chan pollResult
		oldStatus       ResolveState
	}

	pollerState struct {
		// LastResult is the last result this poller received.
		LastResult pollResult
		// LastCycle is true after the resolveID changes.
		// This is used to indicate that when resolveID changes again,
		// the poller should give up if still going.
		LastCycle bool
	}

	// Server copied from server - avoiding coupling to server implemention
	Server struct {
		ClusterName string
		URL         string
	}

	// copied from server
	gdmData struct {
		Deployments []*Deployment
	}

	// ServerListData copied from server - avoiding coupling to server implemention
	ServerListData struct {
		Servers []Server
	}

	// copied from server - avoiding coupling to server implemention
	statusData struct {
		// Deployments is deprecated - there's not a list of intended deployments
		// on each resolve status
		// We still parse it, in case we're talking to an old server.
		// For 1.0 this field should go away.
		Deployments           []*Deployment
		Completed, InProgress *ResolveStatus
	}

	pollResult struct {
		url       string
		stat      ResolveState
		err       error
		resolveID string
	}
)

// PollTimeout is the pause between each polling request to /status.
const PollTimeout = 500 * time.Millisecond

// NewStatusPoller returns a new *StatusPoller.
func NewStatusPoller(cl restful.HTTPClient, rf *ResolveFilter, user User, logs logging.LogSink) *StatusPoller {
	return &StatusPoller{
		HTTPClient:    cl,
		ResolveFilter: rf,
		User:          user,
		logs:          logs,
	}
}

// Wait begins the process of polling for cluster statuses, waits for it to
// complete, and then returns the result, as long as the provided context is not
// cancelled.
func (sp *StatusPoller) Wait(ctx context.Context) (ResolveState, error) {
	var resolveState ResolveState
	var err error
	reportPollerStart(sp.logs, sp)
	done := make(chan struct{})
	go func() {
		resolveState, err = sp.waitForever()
		reportPollerResolved(sp.logs, sp, resolveState, err)
		close(done)
	}()
	select {
	case <-done:
		return resolveState, err
	case <-ctx.Done():
		return resolveState, ctx.Err()
	}
}

func (sp *StatusPoller) waitForever() (ResolveState, error) {
	sp.results = make(chan pollResult)
	// Retrieve the list of servers known to our main server.
	clusters := &ServerListData{}
	if _, err := sp.Retrieve("./servers", nil, clusters, sp.User.HTTPHeaders()); err != nil {
		return ResolveFailed, err
	}

	// Get the up-to-the-moment version of the GDM.
	gdm := &gdmData{}
	if _, err := sp.Retrieve("./gdm", nil, gdm, sp.User.HTTPHeaders()); err != nil {
		return ResolveFailed, err
	}

	// Filter down to the deployments we are interested in.
	deps := NewDeployments(gdm.Deployments...)
	deps = deps.Filter(sp.ResolveFilter.FilterDeployment)
	if deps.Len() == 0 {
		// No deployments match the filter, bail out now.
		messages.ReportLogFieldsMessage("No deployments from /gdm matched", logging.DebugLevel, sp.logs, sp.ResolveFilter)
		return ResolveNotIntended, nil
	}

	// Create a sub-poller for each cluster we are interested in.
	subs, err := sp.subPollers(clusters, deps)
	if err != nil {
		return ResolveNotPolled, err
	}

	return sp.poll(subs), nil
}

func (sp *StatusPoller) subPollers(clusters *ServerListData, deps Deployments) ([]*subPoller, error) {
	subs := []*subPoller{}
	for _, s := range clusters.Servers {
		// skip clusters the user isn't interested in
		if !sp.ResolveFilter.FilterClusterName(s.ClusterName) {
			messages.ReportLogFieldsMessage("Not rquested for polling", logging.ExtraDebug1Level, sp.logs, s.ClusterName)
			continue
		}
		// skip clusters that there's no current intention of deploying into
		if _, intended := deps.Single(func(d *Deployment) bool {
			return d.ClusterName == s.ClusterName
		}); !intended {
			messages.ReportLogFieldsMessage("No intention in GDM for deploy", logging.DebugLevel, sp.logs, s.ClusterName, sp.ResolveFilter)
			continue
		}
		messages.ReportLogFieldsMessage("Starting poller against", logging.DebugLevel, sp.logs, s)
		// Kick off a separate process to issue HTTP requests against this cluster.
		sub, err := newSubPoller(s.ClusterName, s.URL, sp.ResolveFilter, sp.User, sp.logs.Child(s.ClusterName))
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// poll collects updates from each "sub" poller; once they've all crossed the
// TERMINAL threshold, return the "maximum" state reached. We hope for ResolveComplete.
func (sp *StatusPoller) poll(subs []*subPoller) ResolveState {
	if len(subs) == 0 {
		return ResolveNotIntended
	}
	done := make(chan struct{})
	go func() {
		sp.statePerCluster = map[string]*pollerState{}
		for {
			update := <-sp.results
			sp.nextSubStatus(update)

			if sp.finished() {
				close(done)
				return
			}
		}
	}()

	for _, s := range subs {
		go s.start(sp.results, done)
	}

	<-done
	return sp.status
}

func (sp *StatusPoller) nextSubStatus(update pollResult) {
	if lastState, ok := sp.statePerCluster[update.url]; ok {
		if lastState.LastResult.resolveID != "" && lastState.LastResult.resolveID != update.resolveID {
			lastState.LastCycle = true
		}
	} else {
		sp.statePerCluster[update.url] = &pollerState{LastResult: update}
	}
	if sp.statePerCluster[update.url].LastResult != update {
		reportSubreport(sp.logs, sp, update)
	}
	sp.statePerCluster[update.url].LastResult = update
}

func (sp *StatusPoller) finished() bool {
	sp.updateStatus()
	return sp.status >= ResolveTERMINALS
}

func (sp *StatusPoller) updateStatus() {
	currentStatus := sp.status
	sp.status = sp.computeStatus()
	if !(sp.status.String() == sp.oldStatus.String() && sp.status.Prose() == sp.oldStatus.Prose()) {
		reportPollerStatus(sp.logs, sp, currentStatus)
	}
	sp.oldStatus = sp.status
}

func (sp *StatusPoller) computeStatus() ResolveState {
	firstCycleMax := ResolveState(0)
	firstCycleMin := ResolveMAX
	lastCycleMax := ResolveState(0)
	lastCycleMin := ResolveMAX
	for _, s := range sp.statePerCluster {
		if s.LastCycle {
			lastCycleMax = maxStatus(lastCycleMax, s.LastResult.stat)
			lastCycleMin = minStatus(lastCycleMin, s.LastResult.stat)
			if s.LastResult.stat == ResolveNotVersion {
				return ResolveFailed
			}
			if s.LastResult.stat == ResolveNotStarted {
				return ResolveFailed
			}
		} else {
			firstCycleMax = maxStatus(firstCycleMax, s.LastResult.stat)
			firstCycleMin = minStatus(firstCycleMin, s.LastResult.stat)
		}
	}

	// All completed successfully in first cycle.
	if firstCycleMax == ResolveComplete && firstCycleMin == ResolveComplete {
		return ResolveComplete
	}

	// If any poller detects ResolveHTTPFailed, we'll never see another status
	if firstCycleMax == ResolveHTTPFailed || lastCycleMax == ResolveHTTPFailed {
		return ResolveHTTPFailed
	}

	// At least one resolution in second cycle has failed.
	if lastCycleMax == ResolveFailed {
		return ResolveFailed
	}

	// Nothing is in last cycle.
	if lastCycleMax == 0 && lastCycleMin == ResolveMAX {
		return minStatus(firstCycleMax, ResolveInProgress)
	}

	// All in last cycle from this point on.

	// There is at least one resolution still in first cycle.
	if firstCycleMax != 0 && firstCycleMin != ResolveMAX {
		// Report ResolveInProgress because still waiting for
		// certainty.
		return ResolveInProgress
	}

	// All complete.
	if lastCycleMax == ResolveComplete && lastCycleMin == ResolveComplete {
		return ResolveComplete
	}

	// Report at least ResolveInProgress since we are on second resolve
	// and don't want statuses to go backwards.
	return maxStatus(ResolveInProgress, lastCycleMin)
}
