package sous

import (
	"fmt"

	"github.com/samsalisbury/semv"
)

type (
	// Manifests is a collection of Manifest.
	Manifests []*Manifest
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
		Deployments map[string]PartialDeploySpec `validate:"keys=nonempty,values=nonzero"`
	}
	// SourceLocation identifies a directory inside a specific source code repo.
	// Note that the directory has no meaning without the addition of a revision
	// ID. This type is used as a shorthand for deploy manifests, enabling the
	// logical grouping of deploys of different versions of a particular
	// service.
	SourceLocation struct {
		// RepoURL is the URL of a source code repository.
		RepoURL
		// RepoOffset is a relative path to a directory within the repository
		// at RepoURL
		RepoOffset `yaml:",omitempty"`
	}
	// ManifestKind describes the broad category of a piece of software, such as
	// a long-running HTTP service, or a scheduled task, etc. It is used to
	// determine resource sets and contracts that can be run on this
	// application.
	ManifestKind string

	// DeploymentSpecs is a list of DeploymentSpecs.
	DeploymentSpecs []PartialDeploySpec

	// DeploymentSpec is the interface to describe a cluster-wide deployment of
	// an application described by a Manifest. Together with the manifest, one
	// can assemble full Deployments.
	//
	// Unexported fields in DeploymentSpec are not intended to be serialised
	// to/from yaml, but are useful when set internally.
	PartialDeploySpec struct {
		// DeployConfig contains config information for this deployment, see
		// DeployConfig.
		DeployConfig `yaml:"inline"`
		// Version is a semantic version with the following properties:
		//
		//     1. The major/minor/patch/pre-release fields exist as a tag in
		//        the source code repository containing this application.
		//     2. The metadata field is the full revision ID of the commit
		//        which the tag in 1. points to.
		Version semv.Version `validate:"nonzero"`
		// clusterName is the name of the cluster this deployment belongs to. Upon
		// parsing the Manifest, this will be set to the key in
		// Manifests.Deployments which points at this Deployment.
		clusterName string
	}

	// DeployConfig represents the configuration of a deployment's tasks,
	// in a specific cluster. i.e. their resources, environment, and the number
	// of instances.
	DeployConfig struct {
		// Resources represents the resources each instance of this software
		// will be given by the execution environment.
		Resources Resources `validate:"keys=nonempty,values=nonempty"`
		// Env is a list of environment variables to set for each instance of
		// of this deployment. It will be checked for conflict with the
		// definitions found in State.Defs.EnvVars, and if not in conflict
		// assumes the greatest priority.
		Env map[string]string `validate:"keys=nonempty,values=nonempty"`
		// NumInstances is a guide to the number of instances that should be
		// deployed in this cluster, note that the actual number may differ due
		// to decisions made by Sous. If set to zero, Sous will decide how many
		// instances to launch.
		NumInstances int
	}

	// Resources is a mapping of resource name to value, used to provision
	// single instances of an application. It is validated against
	// State.Defs.Resources. The keys must match defined resource names, and the
	// values must parse to the defined types.
	Resources map[string]string
)

func (dc *DeployConfig) String() string {
	return fmt.Sprintf("#%d %+v : %+v", dc.NumInstances, dc.Resources, dc.Env)
}

const (
	// HTTP Service represents an HTTP service which is a long-running process,
	// and listens and responds to HTTP requests.
	ManifestKindService   (ManifestKind) = "http-service"
	ManifestKindWorker    (ManifestKind) = "worker"
	ManifestKindOnDemand  (ManifestKind) = "on-demand"
	ManifestKindScheduled (ManifestKind) = "scheduled"
	ManifestKindOnce      (ManifestKind) = "once"
	// ScheduledJob represents a process which starts on some schedule, and
	// exits when it completes its task.
	ScheduledJob = "scheduled-job"
)
