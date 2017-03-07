package sous

import (
	"time"

	"github.com/pkg/errors"
)

type (
	// StatusPoller polls servers for status.
	StatusPoller struct {
		HTTPClient
		*ResolveFilter
		User User
	}

	subPoller struct {
		HTTPClient
		ClusterName, URL         string
		Deployments              *Deployments
		locationFilter, idFilter *ResolveFilter
		User                     User
	}

	// copied from server - avoiding coupling to server implemention
	server struct {
		ClusterName string
		URL         string
	}

	// copied from server
	gdmData struct {
		Deployments []*Deployment
	}

	// copied from server - avoiding coupling to server implemention
	serverListData struct {
		Servers []server
	}

	// copied from server - avoiding coupling to server implemention
	statusData struct {
		Deployments           []*Deployment
		Completed, InProgress *ResolveStatus
	}

	// A ResolveState reflects the state of the Sous clusters in regard to
	// resolving a particular SourceID.
	ResolveState int

	statPair struct {
		url  string
		stat ResolveState
	}
)

const (
	// ResolveNotPolled is the entry state. It means we haven't received data
	// from a server yet.
	ResolveNotPolled ResolveState = iota
	// ResolveNotStarted conveys the condition that the server is not yet working
	// to resolve the SourceLocation in question. Granted that a manifest update
	// has succeeded, expect that once the current auto-resolve cycle concludes,
	// the resolve-subject GDM will be updated, and we'll move past this state.
	ResolveNotStarted
	// ResolveNotVersion conveys that the server knows the SourceLocation
	// already, but is resolving a different version. Again, expect that on the
	// next auto-resolve cycle we'll move past this state.
	ResolveNotVersion
	// ResolvePendingRequest conveys that, while the server has registered the
	// intent for the current resolve cycle, no request has yet been made to
	// Singularity.
	ResolvePendingRequest
	// ResolveInProgress conveys a resolve action has been taken by the server,
	// which implies that the server's intended version (which we've confirmed is
	// the same as our intended version) is different from the
	// Mesos/Singularity's version.
	ResolveInProgress
	// ResolveErredHTTP  conveys that the HTTP request to the server returned an error
	ResolveErredHTTP
	// ResolveErredRez conveys that the resolving server reported a transient error
	ResolveErredRez
	// ResolveTERMINALS is not a state itself: it demarks resolution states that
	// might proceed from states that are complete
	ResolveTERMINALS
	// ResolveNotIntended indicates that a particular cluster does not intend to
	// deploy the given deployment(s)
	ResolveNotIntended
	// ResolveFailed indicates that a particular cluster is in a failed state
	// regarding resolving the deployments, and that resolution cannot proceed.
	ResolveFailed
	// ResolveTasksStarting is the state when the resolution is complete from
	// Sous' point of view, but awaiting tasks starting in the cluster.
	ResolveTasksStarting
	// ResolveComplete is the success state: the server knows about our intended
	// deployment, and that deployment has returned as having been stable.
	ResolveComplete
	// ResolveMAX is not a state itself: it marks the top end of resolutions. All
	// other states belong before it.
	ResolveMAX
)

// XXX we might consider using go generate with `stringer` (c.f.)
func (rs ResolveState) String() string {
	switch rs {
	default:
		return "unknown (oops)"
	case ResolveNotPolled:
		return "ResolveNotPolled"
	case ResolveNotStarted:
		return "ResolveNotStarted"
	case ResolvePendingRequest:
		return "ResolvePendingRequest"
	case ResolveNotVersion:
		return "ResolveNotVersion"
	case ResolveInProgress:
		return "ResolveInProgress"
	case ResolveErredHTTP:
		return "ResolveErredHTTP"
	case ResolveErredRez:
		return "ResolveErredRez"
	case ResolveTasksStarting:
		return "ResolveTasksStarting"
	case ResolveTERMINALS:
		return "resolve terminal marker - not a real state, received in error?"
	case ResolveNotIntended:
		return "ResolveNotIntended"
	case ResolveFailed:
		return "ResolveFailed"
	case ResolveComplete:
		return "ResolveComplete"
	case ResolveMAX:
		return "resolve maximum marker - not a real state, received in error?"
	}
}

// NewStatusPoller returns a new *StatusPoller.
func NewStatusPoller(cl HTTPClient, rf *ResolveFilter, user User) *StatusPoller {
	return &StatusPoller{
		HTTPClient:    cl,
		ResolveFilter: rf,
		User:          user,
	}
}

func newSubPoller(clusterName, serverURL string, baseFilter *ResolveFilter, user User) (*subPoller, error) {
	cl, err := NewClient(serverURL)
	if err != nil {
		return nil, err
	}

	loc := *baseFilter
	loc.Cluster = ""
	loc.Tag = ""
	loc.Revision = ""

	id := *baseFilter
	id.Cluster = ""

	return &subPoller{
		ClusterName:    clusterName,
		URL:            serverURL,
		HTTPClient:     cl,
		locationFilter: &loc,
		idFilter:       &id,
		User:           user,
	}, nil
}

// Start begins the process of polling for cluster statuses.
func (sp *StatusPoller) Start() (ResolveState, error) {
	clusters := &serverListData{}
	// retrieve the list of servers known to our main server
	if err := sp.Retrieve("./servers", nil, clusters, sp.User); err != nil {
		return ResolveFailed, err
	}
	gdm := &gdmData{}
	// next, get the up-to-the-moment version of the GDM
	if err := sp.Retrieve("./gdm", nil, gdm, sp.User); err != nil {
		return ResolveFailed, err
	}
	deps := NewDeployments(gdm.Deployments...)
	deps = deps.Filter(sp.ResolveFilter.FilterDeployment)
	// if our filter doesn't match anything in the GDM, there's no point continuing polling
	if deps.Len() == 0 {
		return ResolveNotIntended, nil
	}

	subs := []*subPoller{}

	for _, s := range clusters.Servers {
		// skip clusters the user isn't interested in
		if !sp.ResolveFilter.FilterClusterName(s.ClusterName) {
			Log.Vomit.Printf("%s not requested for polling", s.ClusterName)
			continue
		}
		// skip clusters that there's no current intention of deploying into
		if _, intended := deps.Single(func(d *Deployment) bool {
			return d.ClusterName == s.ClusterName
		}); !intended {
			Log.Debug.Printf("No intention in GDM for %s to deploy %s", s.ClusterName, sp.ResolveFilter)
			continue
		}
		Log.Debug.Printf("Starting poller against %v", s)

		// kick of a separate process to issue HTTP requests against this cluster
		sub, err := newSubPoller(s.ClusterName, s.URL, sp.ResolveFilter, sp.User)
		if err != nil {
			return ResolveNotPolled, err
		}
		subs = append(subs, sub)
	}

	return sp.poll(subs), nil
}

// poll collects updates from each "sub" poller; once they've all crossed the
// TERMINAL threshold, return the "maximum" state reached. We hope for ResolveComplete.
func (sp *StatusPoller) poll(subs []*subPoller) ResolveState {
	if len(subs) == 0 {
		return ResolveNotIntended
	}
	collect := make(chan statPair)
	done := make(chan struct{})
	totalStatus := ResolveNotPolled
	go func() {
		pollChans := map[string]ResolveState{}
		for {
			update := <-collect
			pollChans[update.url] = update.stat
			Log.Debug.Printf("%s reports state: %s", update.url, update.stat)
			max := ResolveComplete
			for u, s := range pollChans {
				Log.Vomit.Printf("Current state from %s: %s", u, s)
				if s <= max {
					max = s
				}
			}
			totalStatus = max
			if totalStatus >= ResolveTERMINALS {
				close(done)
				return
			}
		}
	}()

	for _, s := range subs {
		go s.start(collect, done)
	}

	<-done
	return totalStatus
}

// start issues a new /status request every half second, reporting the state as computed.
// c.f. pollOnce.
func (sub *subPoller) start(rs chan statPair, done chan struct{}) {
	rs <- statPair{url: sub.URL, stat: ResolveNotPolled}
	stat := sub.pollOnce()
	rs <- statPair{url: sub.URL, stat: stat}
	ticker := time.NewTicker(time.Second / 2)
	defer ticker.Stop()
	for {
		if stat > ResolveTERMINALS {
			return
		}
		select {
		case <-ticker.C:
			stat = sub.pollOnce()
			rs <- statPair{url: sub.URL, stat: stat}
		case <-done:
			return
		}
	}
}

func (sub *subPoller) pollOnce() ResolveState {
	data := &statusData{}
	if err := sub.Retrieve("./status", nil, data, sub.User); err != nil {
		Log.Debug.Printf("%s: error on GET /status: %s", sub.ClusterName, errors.Cause(err))
		Log.Vomit.Printf("%s: %T %+v", sub.ClusterName, errors.Cause(err), err)
		return ResolveErredHTTP
	}
	deps := NewDeployments(data.Deployments...)
	sub.Deployments = &deps

	return sub.computeState(
		// The serverIntent is the deployment in the GDM when the server started
		// its current resolve sweep.  Note that this might be different from the
		// /gdm query the parent poller collected if, for instance, the resolution
		// started before the most recent update to the GDM.
		sub.serverIntent(),

		// The stable/current statuses are the complete status log collected in
		// the most recent completed resolution, and the live status log being
		// collected by a still-running resolution, if any. We filter the
		// deployment we care about out of those logs.
		data.stableFor(sub.locationFilter),
		data.currentFor(sub.locationFilter),
	)
}

// computeState takes the servers intended deployment, and the stable and
// current DiffResolutions and computes the state of resolution for the
// deployment based on that data.
func (sub *subPoller) computeState(srvIntent *Deployment, stable, current *DiffResolution) ResolveState {
	Log.Debug.Printf("%s reports intent to resolve [%v]", sub.URL, srvIntent)
	Log.Debug.Printf("%s reports stable rez: %v", sub.URL, stable)
	Log.Debug.Printf("%s reports in-progress rez: %v", sub.URL, current)

	// In there's no intent for the deployment in the current resolution, we
	// haven't started on it yet. Remember that we've already determined that the
	// most-recent GDM does have the deployment scheduled for this cluster, so it
	// should be picked up in the next cycle.
	if srvIntent == nil {
		return ResolveNotStarted
	}

	// This is a nuanced distinction from the above: the cluster is in the
	// process of resolving a different version than what we're watching for.
	// Again, if it weren't in the freshest GDM, we wouldn't have gotten here.
	// Next cycle! (note that in both cases, we're likely to poll again several
	// times before that cycle starts.)
	if !sub.idFilter.FilterDeployment(srvIntent) {
		return ResolveNotVersion
	}

	// We prefer the "current" DiffResolution, but the cluster may not have
	// gotten to it yet to put in its log. In that case, we'll consider the most
	// recent stable log.
	if current == nil {
		current = stable
	}

	// If there's no DiffResolution yet for our Deployment, then we're still
	// waiting for a relatively recent change to the GDM to be processed. I think
	// this could only happen in the first attempt to resolve a recent change to
	// the GDM, and only before the cluster has gotten a DiffResolution recorded.
	if current == nil {
		return ResolvePendingRequest
	}

	if current.Error != nil {
		// Certain errors in resolution may clear on their own. (Generally
		// speaking, these are HTTP errors from Singularity which we hope/assume
		// will become successes with enough persistence - i.e. on the next
		// resolution cycle, Singularity will e.g. have finished a pending->running
		// transition and be ready to receive a new Deploy)
		if IsTransientResolveError(current.Error) {
			Log.Debug.Printf("%s: received resolver error %s, retrying", sub.ClusterName, current.Error)
			return ResolveErredRez
		}
		// Other errors are unlikely to clear by themselves. In this case, log the
		// error for operator action, and consider this subpoller done as failed.
		Log.Vomit.Printf("%#v", current)
		Log.Vomit.Printf("%#v", current.Error)
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
		Log.Warn.Printf("Deployment of %s to %s failed: %s", subject, sub.ClusterName, current.Error.String)
		return ResolveFailed
	}

	// In the case where the GDM and ADS deployments are the same, the /status
	// will be described as "unchanged." The upshot is that the most current
	// intend to deploy matches this cluster's current resolver's intend to
	// deploy, and that that matches the deploy that's running. Success!
	if current.Desc == StableDiff {
		return ResolveComplete
	}

	if current.Desc == ComingDiff {
		return ResolveTasksStarting
	}

	return ResolveInProgress
}

func (sub *subPoller) serverIntent() *Deployment {
	Log.Vomit.Printf("Filtering %#v", sub.Deployments)
	Log.Debug.Printf("Filtering with %q", sub.locationFilter)
	dep, exactlyOne := sub.Deployments.Single(sub.locationFilter.FilterDeployment)
	if !exactlyOne {
		Log.Debug.Printf("With %s we didn't match exactly one deployment.", sub.locationFilter)
		return nil
	}
	Log.Vomit.Printf("Matching deployment: %#v", dep)
	return dep
}

func diffResolutionFor(rstat *ResolveStatus, rf *ResolveFilter) *DiffResolution {
	if rstat == nil {
		Log.Vomit.Printf("Status was nil - no match for %s", rf)
		return nil
	}
	rezs := rstat.Log
	for _, rez := range rezs {
		if rf.FilterManifestID(rez.ManifestID) {
			Log.Vomit.Printf("Matching intent for %s: %#v", rf, rez)
			return &rez
		}
	}
	Log.Vomit.Printf("No match for %s in %d entries", rf, len(rezs))
	return nil
}

func (data *statusData) stableFor(rf *ResolveFilter) *DiffResolution {
	return diffResolutionFor(data.Completed, rf)
}

func (data *statusData) currentFor(rf *ResolveFilter) *DiffResolution {
	return diffResolutionFor(data.InProgress, rf)
}
