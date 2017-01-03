package sous

import "time"

type (
	// StatusPoller polls servers for status.
	StatusPoller struct {
		*HTTPClient
		*ResolveFilter
	}

	subPoller struct {
		*HTTPClient
		locationFilter, idFilter *ResolveFilter
	}

	// copied from server - avoiding coupling to server implemention
	server struct {
		URL string
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
		stat resolveState
	}
)

const (
	// ResolveNotPolled is the entry state. It means we haven't received data
	// from a server yet.
	ResolveNotPolled resolveState = iota
	// ResolveNotStarted conveys the condition that the server is not yet working
	// to resolve the SourceLocation in question. Granted that a manifest update
	// has succeeded, expect that once the current auto-resolve cycle concludes,
	// the resolve-subject GDM will be updated, and we'll move past this state.
	ResolveNotStarted
	// ResolvedNotVersion conveys that the server knows the SourceLocation
	// already, but is resolving a different version. Again, expect that on the
	// next auto-resolve cycle we'll move past this state.
	ResolveNotVersion
	// ResolveInProgress conveys a resolve action has been taken by the server,
	// which implies that the server's intended version (which we've confirmed is
	// the same as our intended version) is different from the
	// Mesos/Singularity's version.
	ResolveInProgress
	// ResolveErred conveys that the resolution returned an error. This might be transient.
	ResolveErred
	// ResolveComplete is the success state: the server knows about our intended
	// deployment, and that deployment has returned as having been stable.
	ResolveComplete
)

func newSubPoller(serverURL string, baseFilter *ResolveFilter) (*subPoller, error) {
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
		HTTPClient:     cl,
		locationFilter: &loc,
		idFilter:       &id,
	}, nil
}

// Start begins the process of polling for cluster statuses.
func (sp *StatusPoller) Start() error {
	clusters := &serverListData{}
	if err := sp.Retrieve("./servers", nil, clusters); err != nil {
		return err
	}

	subs := []*subPoller{}

	for _, s := range cluster {
		sub, err := newSubPoller(s.URL)
		if err != nil {
			return err
		}
		subs = append(subs, sub)
	}

	return sp.poll(subs)
}

func (sp *StatusPoller) poll(subs []*subPoller) ResolveState {
	collect := make(chan statPair)
	done := make(chan struct{})
	totalStatus := ResolveNotPolled
	go func() {
		pollChans := map[string]resolveState{}
		for {
			update := <-collect
			pollChans[update.url] = update.stat
			max := ResolveComplete
			for u, s := range pollChans {
				if s <= max {
					max = s
				}
			}
			totalStatus = max
			if totalStatus >= ResolveComplete {
				close(done)
				return
			}
		}
	}()

	for _, s := range subs {
		go sub.start(collect, done)
	}

	<-done
	return totalStatus
}

func (sub *subPoller) start(rs chan statPair, done chan struct{}) {
	rs <- statPair{url: sub.HTTPClient.serverURL, stat: ResolveNotPolled}
	rs <- pollOnce()
	for {
		select {
		case <-time.Tick(time.Second / 2):
			stat := pollOnce()
			rs <- statPair{url: sub.HTTPClient.serverURL, stat: stat}
			if stat >= ResolveComplete {
				return
			}
		case <-done:
			return
		}
	}
}

func (sub *subPoller) pollOnce() resolveState {
	data := &statusData{}
	sub.Retrieve("./status", nil, data)

	return sub.computeState(
		data.intentionFor(sub.locationFilter),
		data.stableFor(sub.locationFilter),
		data.currentFor(sub.locationFilter),
	)
}

func (sub *subPoller) computeState(srvIntent *Deployment, stable, current *DiffResolution) ResolveState {
	Log.Debug.Printf("%s reports intent to resolve %v", sub.HTTPClient.serverURL, srvIntent)
	Log.Debug.Printf("%s reports stable rez: %v", sub.HTTPClient.serverURL, stable)
	Log.Debug.Printf("%s reports in-progress rez: %v", sub.HTTPClient.serverURL, current)

	if srvIntent == nil {
		return ResolveNotStarted
	}

	if !sub.idFilter.FilterDeployment(srvIntent) {
		return ResolveNotVersion
	}

	if current == nil {
		current = stable
	}

	if current.Error != nil {
		return ResolveErred
	}

	if current.Desc == "unchanged" {
		return ResolveComplete
	}

	return ResolveInProgress
}
