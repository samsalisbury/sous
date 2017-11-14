package sous

import (
	"fmt"

	"github.com/pkg/errors"
)

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
	ManifestKindWorker ManifestKind = "worker"
	// ManifestKindOnDemand represents an on-demand service.
	ManifestKindOnDemand ManifestKind = "on-demand"
	// ManifestKindScheduled represents a scheduled task.
	ManifestKindScheduled ManifestKind = "scheduled"
	// ManifestKindOnce represents a one-off job.
	ManifestKindOnce ManifestKind = "once"
	// ScheduledJob represents a process which starts on some schedule, and
	// exits when it completes its task.
	ScheduledJob ManifestKind = "scheduled-job"
)

// Validate returns a list of flaws with this ManifestKind.
func (mk ManifestKind) Validate() []Flaw {
	switch mk {
	default:
		return []Flaw{GenericFlaw{
			Desc: fmt.Sprintf("ManifestKind %q not valid", mk),
			RepairFunc: func() error {
				return errors.Errorf("unable to repair invalid ManifestKind")
			},
		}}
	case ManifestKindService, ManifestKindWorker, ManifestKindOnDemand, ManifestKindScheduled, ManifestKindOnce, ScheduledJob:
		return nil
	}
}
