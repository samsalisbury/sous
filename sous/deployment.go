package sous

import "fmt"

type (
	// Deployments is a collection of Deployment.
	Deployments []*Deployment
	// Deployment is a completely configured deployment of a piece of software.
	// It contains all the data necessary for Sous to create a single
	// deployment, which is a single version of a piece of software, running in
	// a single cluster.
	Deployment struct {
		// DeployConfig contains configuration info for this deployment,
		// including environment variables, resources, suggested instance count.
		DeployConfig `yaml:"inline"`
		// Cluster is the name of the cluster this deployment belongs to. Upon
		// parsing the Manifest, this will be set to the key in
		// Manifests.Deployments which points at this Deployment.
		Cluster string
		// SourceVersion is the precise version of the software to be deployed.
		SourceVersion SourceVersion
		// Owners is a map of named owners of this repository. The type of this
		// field is subject to change.
		Owners OwnerSet
		// Kind is the kind of software that SourceRepo represents.
		Kind ManifestKind

		// Notes collected from the deployment's source
		Annotation
	}

	// DeploymentState is used in a DeploymentIntention to describe the state of
	// the deployment: e.g. whether it's been acheived or not
	DeploymentState uint

	// LogicalSequence is used to order DeploymentIntentions and keep track of a
	// canonical order in which they should be satisfied
	LogicalSequence uint

	// An Annotation stores notes about data available from the source of
	// of a Deployment. For instance, the Id field from the source
	// SingularityRequest for a Deployment can be stored to refer to the source post-diff.
	// They don't participate in equality checks on the deployment
	Annotation struct {
		// RequestID stores the Singularity Request ID that was used for this deployment
		RequestID string
	}

	// DeploymentIntentions represents deployments commanded by a user.
	DeploymentIntentions []DeploymentIntention

	// A DeploymentIntention represents a deployment commanded by a user, possibly not yet acheived
	DeploymentIntention struct {
		Deployment
		// State is the relative state of this intention.
		State DeploymentState

		// The sequence this intention was resolved in - might be e.g. synthesized while walking
		// a git history. This might be left as implicit on the sequence of DIs in a []DI,
		// but if there's a change in storage (i.e. not git), or two single DIs need to be compared,
		// the sequence is useful
		Sequence LogicalSequence
	}

	// A DepName is the name of a deployment
	DepName struct {
		cluster string
		source  SourceLocation
	}

	// OwnerSet collects the names of the owners of a deployment
	OwnerSet map[string]struct{}
)

const (
	// Current means the the deployment is the one currently running
	Current DeploymentState = iota

	// Acheived means that the deployment was realized in infrastructure at some point
	Acheived = iota

	// Waiting means the deployment hasn't yet been acheived
	Waiting = iota

	// PassedOver means that the deployment was received but a different deployment was received before this one could be deployed
	PassedOver = iota
)

// Add adds an owner to an ownerset
func (os OwnerSet) Add(owner string) {
	os[owner] = struct{}{}
}

// Remove removes an owner from an ownerset
func (os OwnerSet) Remove(owner string) {
	delete(os, owner)
}

// Equal returns true if two ownersets contain the same owner names
func (os OwnerSet) Equal(o OwnerSet) bool {
	if len(os) != len(o) {
		return false
	}
	for ownr := range os {
		if _, has := o[ownr]; !has {
			return false
		}
	}

	return true
}

// Add adds a deployment to a Deployments
func (ds *Deployments) Add(d *Deployment) {
	*ds = append(*ds, d)
}

// BuildDeployment constructs a deployment out of a Manifest
func BuildDeployment(m *Manifest, spec PartialDeploySpec, inherit DeploymentSpecs) (*Deployment, error) {
	ownMap := OwnerSet{}
	for i := range m.Owners {
		ownMap.Add(m.Owners[i])
	}
	return &Deployment{
		Cluster: spec.clusterName,
		DeployConfig: DeployConfig{
			Resources:    spec.Resources,
			Env:          spec.Env,
			NumInstances: spec.NumInstances,
		},
		Owners:        ownMap,
		Kind:          m.Kind,
		SourceVersion: m.Source.SourceVersion(spec.Version),
	}, nil
}

func (d *Deployment) String() string {
	return fmt.Sprintf("%s @ %s %s", d.SourceVersion, d.Cluster, d.DeployConfig.String())
}

// Name returns the DepName for a Deployment
func (d *Deployment) Name() DepName {
	return DepName{
		cluster: d.Cluster,
		source:  d.SourceVersion.CanonicalName(),
	}
}

// Equal returns true if two Deployments are equal
func (d *Deployment) Equal(o *Deployment) bool {
	if !(d.Cluster == o.Cluster && d.SourceVersion == o.SourceVersion && d.Kind == o.Kind && len(d.Owners) == len(o.Owners)) {
		return false
	}

	for ownr := range d.Owners {
		if _, has := o.Owners[ownr]; !has {
			return false
		}
	}
	return d.DeployConfig.Equal(o.DeployConfig)
}
