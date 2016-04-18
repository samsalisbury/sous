package sous

type (
	// Deployments is a collection of Deployment.
	Deployments []Deployment
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
		// SourceID is the precise version of the software to be deployed.
		SourceVersion
		// Owners is a list of named owners of this repository. The type of this
		// field is subject to change.
		Owners []string
		// Kind is the kind of software that SourceRepo represents.
		Kind ManifestKind
	}

	DeploymentState uint
	LogicalSequence uint

	// DeploymentIntentions represents deployments commanded by a user.
	DeploymentIntentions []DeploymentIntention
	DeploymentIntention  struct {
		Deployment
		// State is the relative state of this intention.
		State DeploymentState
		// The sequence this intention was resolved in - might be e.g. synthesized while walking
		// a git history. This might be left as implicit on the sequence of DIs in a []DI,
		// but if there's a change in storage (i.e. not git), or two single DIs need to be compared,
		// the sequence is useful
		Sequence LogicalSequence
	}
)

const (
	Current    DeploymentState = iota
	Acheived                   = iota
	Waiting                    = iota
	PassedOver                 = iota
)

func BuildDeployment(m *Manifest, spec PartialDeploySpec, inherit DeploymentSpecs) (*Deployment, error) {
	return &Deployment{
		Cluster: spec.clusterName,
		DeployConfig: DeployConfig{
			Resources:    spec.Resources,
			Env:          spec.Env,
			NumInstances: spec.NumInstances,
		},
		Owners:        m.Owners,
		Kind:          m.Kind,
		SourceVersion: m.Source.NamedVersion(spec.Version),
	}, nil
}
