package sous

import (
	"fmt"
	"io"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
)

type (
	subPoller struct {
		restful.HTTPClient
		ClusterName, URL         string
		locationFilter, idFilter *ResolveFilter
		User                     User
		httpErrorCount           int
		logs                     logging.LogSink
	}
)

func newSubPoller(clusterName, serverURL string, baseFilter *ResolveFilter, user User, logs logging.LogSink) (*subPoller, error) {
	cl, err := restful.NewClient(serverURL, logs.Child("http"))
	if err != nil {
		return nil, err
	}

	loc := *baseFilter
	loc.Cluster = ResolveFieldMatcher{}
	loc.Tag = ResolveFieldMatcher{}
	loc.Revision = ResolveFieldMatcher{}

	id := *baseFilter
	id.Cluster = ResolveFieldMatcher{}

	return &subPoller{
		ClusterName:    clusterName,
		URL:            serverURL,
		HTTPClient:     cl,
		locationFilter: &loc,
		idFilter:       &id,
		User:           user,
		logs:           logs.Child(clusterName),
	}, nil
}

// start issues a new /status request every half second, reporting the state as computed.
// c.f. pollOnce.
func (sub *subPoller) start(rs chan pollResult, done chan struct{}) {
	rs <- pollResult{url: sub.URL, stat: ResolveNotPolled}
	pollResult := sub.pollOnce()
	rs <- pollResult
	ticker := time.NewTicker(PollTimeout)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latest := sub.pollOnce()
			rs <- latest
		case <-done:
			return
		}
	}
}

func (sub *subPoller) result(rs ResolveState, data *statusData, err error) pollResult {
	resolveID := "<none in progress>"
	if data.InProgress != nil {
		resolveID = data.InProgress.Started.String()
	}
	return pollResult{url: sub.URL, stat: rs, resolveID: resolveID, err: err}
}

func (sub *subPoller) pollOnce() pollResult {
	data := &statusData{}
	if _, err := sub.Retrieve("./status", nil, data, sub.User.HTTPHeaders()); err != nil {
		reportDebugSubPollerMessage(fmt.Sprintf("%s: error on GET /status: %s", sub.ClusterName, errors.Cause(err)), sub.logs)
		reportDebugSubPollerMessage(fmt.Sprintf("%s: %T %+v", sub.ClusterName, errors.Cause(err), err), sub.logs)
		sub.httpErrorCount++
		if sub.httpErrorCount > 10 {
			return sub.result(
				ResolveHTTPFailed,
				data,
				fmt.Errorf("more than 10 HTTP errors, giving up; latest error: %s", err),
			)
		}
		return sub.result(ResolveErredHTTP, data, err)
	}
	sub.httpErrorCount = 0

	// This serves to maintain backwards compatibility.
	// XXX One day, remove it.
	if data.Completed != nil && len(data.Completed.Intended) == 0 {
		data.Completed.Intended = data.Deployments
	}
	if data.InProgress != nil && len(data.InProgress.Intended) == 0 {
		data.InProgress.Intended = data.Deployments
	}

	currentState, err := sub.computeState(sub.stateFeatures("in-progress", data.InProgress))

	if currentState == ResolveNotStarted ||
		currentState == ResolveNotVersion ||
		currentState == ResolvePendingRequest {
		state, err := sub.computeState(sub.stateFeatures("completed", data.Completed))
		return sub.result(state, data, err)
	}

	return sub.result(currentState, data, err)
}

func (sub *subPoller) stateFeatures(kind string, rezState *ResolveStatus) (*Deployment, *DiffResolution) {
	current := diffResolutionFor(rezState, sub.locationFilter)
	srvIntent := serverIntent(rezState, sub.locationFilter)
	reportDebugSubPollerMessage(fmt.Sprintf("%s reports %s intent to resolve [%v]", sub.URL, kind, srvIntent), sub.logs)
	reportDebugSubPollerMessage(fmt.Sprintf("%s reports %s rez: %v", sub.URL, kind, current), sub.logs)

	return srvIntent, current
}

func diffResolutionFor(rstat *ResolveStatus, rf *ResolveFilter) *DiffResolution {
	if rstat == nil {
		reportDebugSubPollerMessage(fmt.Sprintf("Status was nil - no match for %s", rf), logging.Log)
		return nil
	}
	rezs := rstat.Log
	for _, rez := range rezs {
		reportDebugSubPollerMessage(fmt.Sprintf("Checking resolution for: %#v(%[1]T)", rez.ManifestID), logging.Log)
		if rf.FilterManifestID(rez.ManifestID) {
			reportDebugSubPollerMessage(fmt.Sprintf("Matching intent for %s: %#v", rf, rez), logging.Log)
			return &rez
		}
	}
	reportDebugSubPollerMessage(fmt.Sprintf("No match for %s in %d entries", rf, len(rezs)), logging.Log)
	return nil
}

func serverIntent(rstat *ResolveStatus, rf *ResolveFilter) *Deployment {
	reportDebugSubPollerMessage(fmt.Sprintf("Filtering with %q", rf), logging.Log)
	if rstat == nil {
		reportDebugSubPollerMessage("Nil resolve status!", logging.Log)
		return nil
	}
	reportDebugSubPollerMessage(fmt.Sprintf("Filtering %s", rstat.Intended), logging.Log)

	var dep *Deployment
	for _, d := range rstat.Intended {
		if rf.FilterDeployment(d) {
			if dep != nil {
				reportDebugSubPollerMessage(fmt.Sprintf("With %s we didn't match exactly one deployment.", rf), logging.Log)
				return nil
			}
			dep = d
		}
	}
	reportDebugSubPollerMessage(fmt.Sprintf("Filtering found %s", dep), logging.Log)
	return dep
}

// computeState takes the servers intended deployment, and the stable and
// current DiffResolutions and computes the state of resolution for the
// deployment based on that data.
func (sub *subPoller) computeState(srvIntent *Deployment, current *DiffResolution) (ResolveState, error) {
	// In there's no intent for the deployment in the current resolution, we
	// haven't started on it yet. Remember that we've already determined that the
	// most-recent GDM does have the deployment scheduled for this cluster, so it
	// should be picked up in the next cycle.
	if srvIntent == nil {
		return ResolveNotStarted, nil
	}

	// This is a nuanced distinction from the above: the cluster is in the
	// process of resolving a different version than what we're watching for.
	// Again, if it weren't in the freshest GDM, we wouldn't have gotten here.
	// Next cycle! (note that in both cases, we're likely to poll again several
	// times before that cycle starts.)
	if !sub.idFilter.FilterDeployment(srvIntent) {
		return ResolveNotVersion, nil
	}

	// If there's no DiffResolution yet for our Deployment, then we're still
	// waiting for a relatively recent change to the GDM to be processed. I think
	// this could only happen in the first attempt to resolve a recent change to
	// the GDM, and only before the cluster has gotten a DiffResolution recorded.
	if current == nil {
		return ResolvePendingRequest, nil
	}

	if current.Error != nil {
		// Certain errors in resolution may clear on their own. (Generally
		// speaking, these are HTTP errors from Singularity which we hope/assume
		// will become successes with enough persistence - i.e. on the next
		// resolution cycle, Singularity will e.g. have finished a pending->running
		// transition and be ready to receive a new Deploy)
		if IsTransientResolveError(current.Error) {
			reportDebugSubPollerMessage(fmt.Sprintf("%s: received resolver error %s, retrying", sub.ClusterName, current.Error), sub.logs)
			return ResolveErredRez, current.Error
		}
		// Other errors are unlikely to clear by themselves. In this case, log the
		// error for operator action, and consider this subpoller done as failed.
		reportDebugSubPollerMessage(fmt.Sprintf("%#v", current), sub.logs)
		reportDebugSubPollerMessage(fmt.Sprintf("%#v", current.Error), sub.logs)

		subject := ""
		if sub.locationFilter == nil {
			subject = "<no filter defined>"
		} else {
			sourceLocation, ok := sub.locationFilter.SourceLocation()
			if ok {
				subject = sourceLocation.String()
			} else {
				subject = sub.locationFilter.String()
			}
		}

		reportSubPollerMessage(fmt.Sprintf("Deployment of %s to %s failed: %s", subject, sub.ClusterName, current.Error.String), sub.logs)
		return ResolveFailed, current.Error
	}

	// In the case where the GDM and ADS deployments are the same, the /status
	// will be described as "unchanged." The upshot is that the most current
	// intend to deploy matches this cluster's current resolver's intend to
	// deploy, and that that matches the deploy that's running. Success!
	if current.Desc == StableDiff {
		return ResolveComplete, nil
	}

	if current.Desc == ComingDiff {
		return ResolveTasksStarting, nil
	}

	return ResolveInProgress, nil
}

type subPollerMessage struct {
	logging.CallerInfo
	msg          string
	isDebugMsg   bool
	isConsoleMsg bool
	isErrorMsg   bool
}

func reportErrorSubPollerMessage(err error, log logging.LogSink) {
	msg := "<missing error>"
	if err != nil {
		msg = err.Error()
	}
	reportSubPollerMessage(msg, log, false, false, true)
}

func reportDebugSubPollerMessage(msg string, log logging.LogSink) {
	reportSubPollerMessage(msg, log, true)
}

func reportConsoleSubPollerMessage(msg string, log logging.LogSink) {
	reportSubPollerMessage(msg, log, false, true)
}

func reportSubPollerMessage(msg string, log logging.LogSink, flags ...bool) {
	debugStmt := false
	consoleMsg := false
	errorMsg := false
	if len(flags) > 0 {
		debugStmt = flags[0]
		if len(flags) > 1 {
			consoleMsg = flags[1]
		}
		if len(flags) > 2 {
			errorMsg = flags[2]
		}
	}

	msgLog := subPollerMessage{
		msg:          msg,
		CallerInfo:   logging.GetCallerInfo(logging.NotHere()),
		isDebugMsg:   debugStmt,
		isConsoleMsg: consoleMsg,
		isErrorMsg:   errorMsg,
	}
	logging.Deliver(log, msgLog)
}

func (msg subPollerMessage) WriteToConsole(console io.Writer) {
	if msg.isConsoleMsg {
		fmt.Fprintf(console, "%s\n", msg.composeMsg())
	}
}

func (msg subPollerMessage) DefaultLevel() logging.Level {
	level := logging.WarningLevel
	if msg.isDebugMsg {
		level = logging.DebugLevel
	}
	if msg.isErrorMsg {
		level = logging.WarningLevel
	}
	return level
}

func (msg subPollerMessage) Message() string {
	return msg.composeMsg()
}

func (msg subPollerMessage) composeMsg() string {
	return msg.msg
}

func (msg subPollerMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousGenericV1)
	msg.CallerInfo.EachField(f)
}
