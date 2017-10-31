package sous

// A ResolveState reflects the state of the Sous clusters in regard to
// resolving a particular SourceID.
type ResolveState int

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
	// ResolveTasksStarting is the state when the resolution is complete from
	// Sous' point of view, but awaiting tasks starting in the cluster.
	ResolveTasksStarting
	// ResolveErredHTTP  conveys that the HTTP request to the server returned an error
	ResolveErredHTTP
	// ResolveErredRez conveys that the resolving server reported a transient error
	ResolveErredRez
	// ResolveNotIntended indicates that a particular cluster does not intend to
	// deploy the given deployment(s)
	ResolveNotIntended
	// ResolveComplete is the success state: the server knows about our intended
	// deployment, and that deployment has returned as having been stable.
	ResolveComplete
	// ResolveFailed indicates that a particular cluster is in a failed state
	// regarding resolving the deployments, and that resolution cannot proceed.
	ResolveFailed
	// ResolveHTTPFailed indicates that the sous server in a particular cluster
	// has returned HTTP errors 10 consequetive times and is assumed down
	ResolveHTTPFailed
	// ResolveMAX is not a state itself: it marks the top end of resolutions. All
	// other states belong before it.
	ResolveMAX

	// ResolveTERMINALS is not a state itself: it demarks resolution states that
	// might proceed from states that are complete
	ResolveTERMINALS = ResolveNotIntended
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
	case ResolveNotIntended:
		return "ResolveNotIntended"
	case ResolveFailed:
		return "ResolveFailed"
	case ResolveHTTPFailed:
		return "ResolveHTTPFailed"
	case ResolveComplete:
		return "ResolveComplete"
	case ResolveMAX:
		return "resolve maximum marker - not a real state, received in error?"
	}
}

// Prose returns a string that explains what the state means.
func (rs ResolveState) Prose() string {
	switch rs {
	default:
		return "unknown (oops)"
	case ResolveNotPolled:
		return "No data from server yet"
	case ResolveNotStarted:
		return "Waiting for server to begin resolution"
	case ResolvePendingRequest:
		return "Waiting for request to be made to cluster"
	case ResolveNotVersion:
		return "Waiting for server to acknowledge new version"
	case ResolveInProgress:
		return "Waiting for cluster to acknowledge deploy request"
	case ResolveErredHTTP:
		return "HTTP request to Sous server errored"
	case ResolveErredRez:
		return "Transient error on server, retrying"
	case ResolveTasksStarting:
		return "Cluster has accepted deploy request, awaiting service boot"
	case ResolveNotIntended:
		return "Sous server does not intend to deploy this version - probably another deploy request received"
	case ResolveFailed:
		return "The attempt to resolve this service failed"
	case ResolveHTTPFailed:
		return "HTTP connection to the Sous server has failed"
	case ResolveComplete:
		return "Deployment resolution is complete"
	case ResolveMAX:
		return "resolve maximum marker - not a real state, received in error?"
	}
}

func minStatus(a, b ResolveState) ResolveState {
	if a < b {
		return a
	}
	return b
}

func maxStatus(a, b ResolveState) ResolveState {
	if a > b {
		return a
	}
	return b
}
