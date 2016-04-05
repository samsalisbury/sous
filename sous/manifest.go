package sous

type (
	// Manifests is a collection of Manifest
	Manifests []Manifest
	// Manifest represents all the global deployments of a single application.
	Manifest struct {
		// SourceRepo is the canonical name of the source code repository
		// containing the code for this application.
		SourceRepo CanonicalPackageName
		// Owners is a list of named owners of this repository. The type of this
		// field is subject to change.
		Owners []string
		// Kind is the kind of software that SourceRepo represents.
		Kind ManifestKind
		// Deployments is a list of deployments, one entry per cluster, as
		// defined in State.Defs.Clusters
		Deployments map[string]Deployment
	}
	// ManifestKind describes the broad category of a piece of software, such as
	// a long-running HTTP service, or a scheduled task, etc.
	ManifestKind string
	// CanonicalPackageName is the canonical name of a source code repository.
	CanonicalPackageName string
	// Deployment is a configured deployment of a piece of software.
	Deployment struct {
		// Cluster is the name of the cluster this deployment belongs to. Upon
		// parsing the Manifest, this will be set to the key in
		// Manifests.Deployments which points at this Deployment.
		Cluster string
		// Resources represents the resources each instance of this software
		// will be given by the execution environment.
		Resources Resources
		// SourceTag is the source code tag whose artifacts should be deployed.
		SourceTag,
		// SourceRevision is the revision ID of the SourceTag, it will be
		// checked at build time to make sure the tag has not been changed.
		SourceRevision string
		// Env is a list of environment variables to set for each instance of
		// of this deployment. It will be checked for conflict with the
		// definitions found in State.Defs.EnvVars, and if not in conflict
		// assumes the greatest priority.
		Env map[string]string
		// NumInstances is a guide to the number of instances that should be
		// deployed in this cluster, note that the actual number may differ due
		// to decisions made by Sous.
		NumInstances int
	}
	// Resources is a mapping of resource name to value, used to provision
	// single instances of an application. It is validated against
	// State.Defs.Resources. The keys must match defined resource names, and the
	// values must parse to the defined types.
	Resources map[string]string
)

const (
	// HTTP Service represents an HTTP service which is a long-running process,
	// and listens and responds to HTTP requests.
	HTTPService (ManifestKind) = "http-service"
	// ScheduledJob represents a process which starts on some schedule, and
	// exits when it completes its task.
	ScheduledJob = "scheduled-job"
)
