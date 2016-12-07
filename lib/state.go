package sous

import (
	"strings"

	"github.com/pkg/errors"
)

type (
	// State contains the mutable state of an organisation's deployments.
	// State is also known as the "Global Deploy Manifest" or GDM.
	State struct {
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
		Resources FieldDefinitions
		// Metadata contains the definitions for metadata fields
		Metadata FieldDefinitions
	}

	// EnvDefs is a collection of EnvDef
	EnvDefs []EnvDef
	// EnvDef is an environment variable definition.
	EnvDef struct {
		Name, Desc, Scope string
		Type              VarType
	}

	// FieldDefinitions is just a type alias for a slice of FieldDefinition-s
	FieldDefinitions []FieldDefinition

	// A FieldDefinition describes the requirements for a Metadata field.
	FieldDefinition struct {
		Name string
		// Type is the type of value used to represent quantities or instances
		// of this resource, e.g. MemorySize, Float, or Int (not yet implemented).
		Type VarType

		// Default adds a GDM wide default for a key.
		// It's assumed that if this is left empty, the field must be set
		Default string `yaml:",omitempty"`

		// If the zero value is the intended default value for a field,
		// you can mark it Optional: true.
		Optional bool `yaml:",omitempty"`
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

// Clone returns a deep copy of this State.
func (s State) Clone() *State {
	s.Manifests = s.Manifests.Clone()
	s.Defs = s.Defs.Clone()
	return &s
}

// Clone returns a deep copy of this Defs.
func (d Defs) Clone() Defs {
	d.Clusters = d.Clusters.Clone()
	d.EnvVars = d.EnvVars.Clone()
	d.Resources = d.Resources.Clone()
	d.Metadata = d.Metadata.Clone()
	return d
}

// Clone returns a deep copy of this Clusters.
func (cs Clusters) Clone() Clusters {
	c := make(Clusters, len(cs))
	for name, cluster := range cs {
		c[name] = cluster.Clone()
	}
	return c
}

// Names returns a slice of names of these clusters.
func (cs Clusters) Names() []string {
	names := make([]string, len(cs))
	i := 0
	for name := range cs {
		names[i] = name
		i++
	}
	return names
}

// Clone returns a deep copy of this Cluster.
func (c Cluster) Clone() *Cluster {
	allowedAdvisories := make([]string, len(c.AllowedAdvisories))
	copy(allowedAdvisories, c.AllowedAdvisories)
	c.AllowedAdvisories = allowedAdvisories
	return &c
}

// Clone returns a deep copy of this EnvDefs.
func (evs EnvDefs) Clone() EnvDefs {
	e := make(EnvDefs, len(evs))
	copy(e, evs)
	return e
}

// Clone returns a deep copy of this ResDefs.
func (rdf FieldDefinitions) Clone() FieldDefinitions {
	r := make(FieldDefinitions, len(rdf))
	copy(r, rdf)
	return r
}

// ClusterMap returns the nicknames for all the clusters referred to in this state
// paired with the URL for the named cluster
func (s *State) ClusterMap() map[string]string {
	m := make(map[string]string, len(s.Defs.Clusters))
	for name, cluster := range s.Defs.Clusters {
		m[name] = cluster.BaseURL
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

// Validate implements Flawed for State
func (s *State) Validate() []Flaw {
	var flaws []Flaw

	for _, manifest := range s.Manifests.Snapshot() {
		flaws = append(flaws, manifest.Validate()...)
	}

	for _, f := range flaws {
		f.AddContext("state", s)
	}
	return flaws
}

// Repair implements Flawed for State
func (s *State) Repair(fs []Flaw) error {
	return errors.Errorf("Can't do nuffin with flaws yet")
}

// UpdateDeployments upserts ds into the State
func (s *State) UpdateDeployments(ds ...*Deployment) error {
	stateDeps, err := s.Deployments()
	if err != nil {
		return err
	}

	for _, d := range ds {
		stateDeps.Set(d.ID(), d)
	}

	newManifests, err := stateDeps.Manifests(s.Defs)
	if err != nil {
		return err
	}

	s.Manifests = newManifests
	return nil
}

func (cs Clusters) String() string {
	var clusterNames []string
	for clusterName := range cs {
		clusterNames = append(clusterNames, clusterName)
	}
	return strings.Join(clusterNames, ", ")
}
