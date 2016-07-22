package sous

import "path/filepath"

type (
	// Manifests is a collection of Manifest.
	Manifests map[string]*Manifest

	// Manifest is a minimal representation of the global deployment state of
	// a particular named application. It is designed to be written and read by
	// humans as-is, and expanded into full Deployments internally. It is a DTO,
	// which can be stored in YAML files.
	//
	// Manifest has a direct two-way mapping to/from Deployments.
	Manifest struct {
		// Source is the location of the source code for this piece of software.
		Source SourceLocation `validate:"nonzero"`
		// Owners is a list of named owners of this repository. The type of this
		// field is subject to change.
		Owners []string
		// Kind is the kind of software that SourceRepo represents.
		Kind ManifestKind `validate:"nonzero"`
		// Deployments is a map of cluster names to DeploymentSpecs
		Deployments DeploySpecs `validate:"keys=nonempty,values=nonzero"`
	}

	// ManifestKind describes the broad category of a piece of software, such as
	// a long-running HTTP service, or a scheduled task, etc. It is used to
	// determine resource sets and contracts that can be run on this
	// application.
	ManifestKind string

	// Env is a mapping of environment variable name to value, used to provision
	// single instances of an application.
	Env map[string]string
)

// FileLocation returns the path that the manifest should be saved to.
func (m *Manifest) FileLocation() string {
	return filepath.Join(string(m.Source.RepoURL), string(m.Source.RepoOffset))
}

const (
	// ManifestKindService represents an HTTP service which is a long-running process,
	// and listens and responds to HTTP requests.
	ManifestKindService (ManifestKind) = "http-service"
	// ManifestKindWorker represents a worker process.
	ManifestKindWorker (ManifestKind) = "worker"
	// ManifestKindOnDemand represents an on-demand service.
	ManifestKindOnDemand (ManifestKind) = "on-demand"
	// ManifestKindScheduled represents a scheduled task.
	ManifestKindScheduled (ManifestKind) = "scheduled"
	// ManifestKindOnce represents a one-off job.
	ManifestKindOnce (ManifestKind) = "once"
	// ScheduledJob represents a process which starts on some schedule, and
	// exits when it completes its task.
	ScheduledJob = "scheduled-job"
)

// Equal compares Envs
func (e Env) Equal(o Env) bool {
	if len(e) != len(o) {
		return false
	}

	for name, value := range e {
		if ov, ok := o[name]; !ok || ov != value {
			return false
		}
	}
	return true
}
