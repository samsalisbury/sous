package sous

import "path/filepath"

//go:generate ggen cmap.CMap(cmap.go) sous.Manifests(manifests.go) CMKey:ManifestID Value:*Manifest

type (
	// Manifest is a minimal representation of the global deployment state of
	// a particular named application. It is designed to be written and read by
	// humans as-is, and expanded into full Deployments internally. It is a DTO,
	// which can be stored in YAML files.
	//
	// Manifest has a direct two-way mapping to/from Deployments.
	Manifest struct {
		// Source is the location of the source code for this piece of software.
		Source SourceLocation `validate:"nonzero"`
		// Flavor is an optional string, used to allow a single SourceLocation
		// to have multiple deployments defined per cluster. The empty Flavor
		// is perfectly valid. The pair (SourceLocation, Flavor) identifies a
		// manifest.
		Flavor string `yaml:",omitempty"`
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
)

// ID returns the SourceLocation.
func (m Manifest) ID() ManifestID {
	return ManifestID{Source: m.Source, Flavor: m.Flavor}
}

// SetID sets the Source and Flavor fields of this Manifest to those of the
// supplied ManifestID.
func (m *Manifest) SetID(mid ManifestID) {
	m.Source = mid.Source
	m.Flavor = mid.Flavor
}

// FileLocation returns the path that the manifest should be saved to.
func (m *Manifest) FileLocation() string {
	return filepath.Join(string(m.Source.Repo), string(m.Source.Dir))
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

// Equal returns true iff o is equal to m.
func (m *Manifest) Equal(o *Manifest) bool {
	if m == o {
		return true
	}
	if m.Source != o.Source {
		return false
	}
	if m.Kind != o.Kind {
		return false
	}
	if len(m.Owners) != len(o.Owners) {
		return false
	}
	for i, owner := range m.Owners {
		if o.Owners[i] != owner {
			return false
		}
	}
	if len(m.Deployments) != len(o.Deployments) {
		return false
	}
	for clusterName, deploySpec := range m.Deployments {
		if !o.Deployments[clusterName].Equal(deploySpec) {
			return false
		}
	}
	return true
}
