package sous

// ManifestKind describes the broad category of a piece of software, such as
// a long-running HTTP service, or a scheduled task, etc. It is used to
// determine resource sets and contracts that can be run on this
// application.
type ManifestKind string

const (
	// ManifestKindService represents an HTTP service which is a long-running process,
	// and listens and responds to HTTP requests.
	ManifestKindService ManifestKind = "http-service"
	// ManifestKindWorker represents a worker process.
	ManifestKindWorker = "worker"
	// ManifestKindOnDemand represents an on-demand service.
	ManifestKindOnDemand = "on-demand"
	// ManifestKindScheduled represents a scheduled task.
	ManifestKindScheduled = "scheduled"
	// ManifestKindOnce represents a one-off job.
	ManifestKindOnce = "once"
	// ScheduledJob represents a process which starts on some schedule, and
	// exits when it completes its task.
	ScheduledJob = "scheduled-job"
)
