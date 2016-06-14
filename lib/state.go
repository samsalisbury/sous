package sous

import (
	"github.com/opentable/sous/util/hy"
	"github.com/opentable/sous/util/yaml"
)

type (
	// State contains the mutable state of an organisation's deployments.
	// State is also known as the "Global Deploy Manifest" or GDM.
	State struct {
		// Defs contains global definitions for this organisation.
		Defs Defs `hy:"defs.yaml"`
		// Manifests contains a mapping of source code repositories to global
		// deployment configurations for artifacts built using that source code.
		Manifests Manifests `hy:"manifests/**"`
	}

	// Defs holds definitions for organisation-level objects.
	Defs struct {
		// DockerRepo is the host:port (no schema) to connect to the Docker repository
		DockerRepo string
		// Clusters is a collection of logical deployment environments.
		Clusters Clusters `hy:"clusters/"`
		// EnvVars contains definitions for global environment variables.
		EnvVars EnvDefs
		// Resources contains definitions for resource types available to
		// deployment manifests.
		Resources ResDefs
	}

	// EnvDefs is a collection of EnvDef
	EnvDefs []EnvDef
	// EnvDef is an environment variable definition.
	EnvDef struct {
		Name, Desc, Scope string
		Type              VarType
	}

	// ResDefs is a collection of ResDef.
	ResDefs []ResDef
	// ResDef is a resource type definition.
	ResDef struct {
		// Name is the name of the resource, e.g. "Memory", "CPU", "NumPorts"
		Name string
		// Type is the type of value used to represent quantities or instances
		// of this resource, e.g. MemorySize, Float, or Int (not yet implemented).
		Type VarType
	}

	// Clusters is a collection of Cluster
	Clusters map[string]Cluster
	// Cluster is a logical deployment target, often named for its region,
	// purpose, etc.
	Cluster struct {
		// Name is the unique name of this cluster.
		Name string `hy:",filename"`
		// Kind is the kind of cluster. Currently the only legal value is
		// "singularity"
		Kind string
		// BaseURL is the main entrypoint URL for interacting with this cluster.
		BaseURL string
		// Env is the default environment for all deployments in this region.
		Env EnvDefaults
	}

	// EnvDefaults is a list of named environment variables along with their values.
	EnvDefaults map[string]Var
	// Var is a strongly typed string for use in environment variables and YAML
	// files. It will implement sane YAML marshalling and unmarshalling. (Not
	// yet implemented.)
	Var string
	// VarType represents the type of a Var (not yet implemented).
	VarType string
)

// LoadState loads the state from a directory
func LoadState(dir string) (st State, err error) {
	u := hy.NewUnmarshaler(yaml.Unmarshal)
	err = u.Unmarshal(dir, &st)
	return
}

// BaseURLs returns the urls for all the clusters referred to in this state
func (st *State) BaseURLs() []string {
	urls := make([]string, 0, len(st.Defs.Clusters))
	for _, cl := range st.Defs.Clusters {
		urls = append(urls, cl.BaseURL)
	}
	return urls
}
