package sous

import (
	"math"
	"path/filepath"
	"strconv"

	"fmt"

	"github.com/samsalisbury/semv"
)

type (
	// Manifests is a collection of Manifest.
	Manifests map[string]*Manifest

	// DeploySpecs is a collection of Deployments associated with a manifest
	DeploySpecs map[string]PartialDeploySpec

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
	// SourceLocation identifies a directory inside a specific source code repo.
	// Note that the directory has no meaning without the addition of a revision
	// ID. This type is used as a shorthand for deploy manifests, enabling the
	// logical grouping of deploys of different versions of a particular
	// service.
	SourceLocation struct {
		// RepoURL is the URL of a source code repository.
		RepoURL RepoURL
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

	// PartialDeploySpec is the interface to describe a cluster-wide deployment
	// of an application described by a Manifest. Together with the manifest,
	// one can assemble full Deployments.
	//
	// Unexported fields in DeploymentSpec are not intended to be serialised
	// to/from yaml, but are useful when set internally.
	PartialDeploySpec struct {
		// DeployConfig contains config information for this deployment, see
		// DeployConfig.
		DeployConfig `yaml:",inline"`
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
		Resources Resources `yaml:",omitempty" validate:"keys=nonempty,values=nonempty"`
		// Env is a list of environment variables to set for each instance of
		// of this deployment. It will be checked for conflict with the
		// definitions found in State.Defs.EnvVars, and if not in conflict
		// assumes the greatest priority.
		Args []string `yaml:",omitempty" validate:"values=nonempty"`
		Env  `yaml:",omitempty" validate:"keys=nonempty,values=nonempty"`
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

	// Env is a mapping of environment variable name to value, used to provision
	// single instances of an application.
	Env map[string]string
)

func (m *Manifest) FileLocation() string {
	return filepath.Join(string(m.Source.RepoURL), string(m.Source.RepoOffset))
}

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

func (dc *DeployConfig) Equal(o DeployConfig) bool {
	Log.Debug.Printf("%+ v ?= %+ v", dc, o)
	return (dc.NumInstances == o.NumInstances && dc.Env.Equal(o.Env) && dc.Resources.Equal(o.Resources))
}

// SingMap produces a dtoMap appropriate for building a Singularity
// dto.Resources struct from
func (r Resources) SingMap() dtoMap {
	return dtoMap{
		"Cpus":     r.cpus(),
		"MemoryMb": r.memory(),
		"NumPorts": int32(r.ports()),
	}
}

func (r Resources) cpus() float64 {
	cpuStr, present := r["cpus"]
	cpus, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		cpus = 0.1
		if present {
			Log.Warn.Printf("Could not parse value: '%s' for cpus as a float, using default: %f", cpuStr, cpus)
		} else {
			Log.Info.Printf("Using default value for cpus: %f", cpus)
		}
	}
	return cpus
}

func (r Resources) memory() float64 {
	memStr, present := r["memory"]
	memory, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		memory = 100
		if present {
			Log.Warn.Printf("Could not parse value: '%s' for memory as an int, using default: %f", memStr, memory)
		} else {
			Log.Info.Printf("Using default value for memory: %f", memory)
		}
	}
	return memory
}

func (r Resources) ports() int32 {
	portStr, present := r["ports"]
	ports, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		ports = 1
		if present {
			Log.Warn.Printf("Could not parse value: '%s' for ports as a int, using default: %d", portStr, ports)
		} else {
			Log.Info.Printf("Using default value for ports: %d", ports)
		}
	}
	return int32(ports)
}

// Equal checks equivalence between resource maps
func (r Resources) Equal(o Resources) bool {
	Log.Debug.Printf("Comparing resources: %+ v ?= %+ v", r, o)
	if len(r) != len(o) {
		Log.Debug.Println("Lengths differ")
		return false
	}

	if r.ports() != o.ports() {
		Log.Debug.Println("Ports differ")
		return false
	}

	if math.Abs(r.cpus()-o.cpus()) > 0.001 {
		Log.Debug.Println("Cpus differ")
		return false
	}

	if math.Abs(r.memory()-o.memory()) > 0.001 {
		Log.Debug.Println("Memory differ")
		return false
	}

	return true
}

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
