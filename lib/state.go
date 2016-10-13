package sous

type (
	// State contains the mutable state of an organisation's deployments.
	// State is also known as the "Global Deploy Manifest" or GDM.
	State struct {
		singleCluster string
		// Defs contains global definitions for this organisation.
		Defs Defs `hy:"defs"`
		// Manifests contains a mapping of source code repositories to global
		// deployment configurations for artifacts built using that source code.
		Manifests Manifests `hy:"manifests/"`
	}

	// Defs holds definitions for organisation-level objects.
	Defs struct {
		// DockerRepo is the host:port (no schema) to connect to the Docker repository
		DockerRepo string
		// Clusters is a collection of logical deployment environments.
		Clusters Clusters
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
	Clusters map[string]*Cluster
	// Cluster is a logical deployment target, often named for its region,
	// purpose, etc.
	Cluster struct {
		// Name is the unique name of this cluster.
		Name string
		// Kind is the kind of cluster. Currently the only legal value is
		// "singularity"
		Kind string
		// BaseURL is the main entrypoint URL for interacting with this cluster.
		BaseURL string
		// Env is the default environment for all deployments in this region.
		Env EnvDefaults
		// AllowedAdvisories lists the artifact advisories which are permissible in
		// this cluster
		AllowedAdvisories []string
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

// NewState returns a valid empty state.
func NewState() *State {
	return &State{
		Manifests: NewManifests(),
	}
}

func (s *State) Clone() (c *State) {
	c = new(State)
	*c = *s
	c.Manifests = NewManifests()
	for _, v := range s.Manifests.Snapshot() {
		c.Manifests.Add(v.Clone())
	}
	c.Defs.Clusters = make(Clusters)
	for k, v := range s.Defs.Clusters {
		c.Defs.Clusters[k] = new(Cluster)
		*c.Defs.Clusters[k] = *v
	}
	c.Defs.EnvVars = make(EnvDefs, len(s.Defs.EnvVars))
	copy(s.Defs.EnvVars, c.Defs.EnvVars)
	c.Defs.Resources = make(ResDefs, len(s.Defs.Resources))
	copy(s.Defs.Resources, c.Defs.Resources)
	return
}

// OnlyCluster sets a contraint on the State such that it will only consider a particular cluster
func (s *State) OnlyCluster(nick string) {
	s.singleCluster = nick
}

// ClusterMap returns the nicknames for all the clusters referred to in this state
// paired with the URL for the named cluster
func (s *State) ClusterMap() map[string]string {
	m := make(map[string]string, len(s.Defs.Clusters))
	for name, cluster := range s.Defs.Clusters {
		if s.singleCluster == "" || s.singleCluster == name {
			m[name] = cluster.BaseURL
		}
	}
	return m
}

// BaseURLs returns the urls for all the clusters referred to in this state
// XXX - deprecate/remove
func (s *State) BaseURLs() []string {
	urls := make([]string, 0, len(s.Defs.Clusters))
	for _, cluster := range s.Defs.Clusters {
		urls = append(urls, cluster.BaseURL)
	}
	return urls
}
