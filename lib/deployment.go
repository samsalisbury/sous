package sous

//go:generate ggen cmap.CMap(cmap.go) sous.Deployments(deployments.go) CMKey:DeployID Value:*Deployment

import (
	"fmt"
	"strings"
)

type (
	// Deployment is a completely configured deployment of a piece of software.
	// It contains all the data necessary for Sous to create a single
	// deployment, which is a single version of a piece of software, running in
	// a single cluster.
	Deployment struct {
		// DeployConfig contains configuration info for this deployment,
		// including environment variables, resources, suggested instance count.
		DeployConfig `yaml:"inline"`
		// ClusterNickname is the human name for a cluster - it's taken from the
		// hash key that defines the cluster and is used in manifests to configure
		// cluster-local deployment config.
		ClusterName string
		// Cluster is the name of the cluster this deployment belongs to. Upon
		// parsing the Manifest, this will be set to the key in
		// Manifests.Deployments which points at this Deployment.
		Cluster *Cluster
		// SourceID is the precise version of the software to be deployed.
		SourceID SourceID
		// Owners is a map of named owners of this repository. The type of this
		// field is subject to change.
		Owners OwnerSet
		// Kind is the kind of software that SourceRepo represents.
		Kind ManifestKind

		// Volumes enumerates the volume mappings required.
		Volumes Volumes

		// Notes collected from the deployment's source.
		Annotation
	}

	// An Annotation stores notes about data available from the source of of a
	// Deployment. For instance, the Id field from the source SingularityRequest
	// for a Deployment can be stored to refer to the source post-diff.  They
	// don't participate in equality checks on the deployment.
	Annotation struct {
		// RequestID stores the Singularity Request ID that was used for this
		// deployment.
		RequestID string
	}

	// DeploymentPredicate takes a *Deployment and returns true if the
	// deployment matches the predicate. Used by Filter to select a subset of a
	// Deployments.
	DeploymentPredicate func(*Deployment) bool

	// A DeployID identifies a deployment.
	DeployID struct {
		Cluster string
		Source  SourceLocation
	}
)

func (d *Deployment) String() string {
	return fmt.Sprintf("%s @ %s %s", d.SourceID, d.Cluster, d.DeployConfig.String())
}

// ID returns the DeployID of this deployment.
func (d *Deployment) ID() DeployID {
	return DeployID{
		Source:  d.SourceID.Location(),
		Cluster: d.ClusterName,
	}
}

// TabbedDeploymentHeaders returns the names of the fields for Tabbed, suitable
// for use with text/tabwriter.
func TabbedDeploymentHeaders() string {
	return "Cluster\t" +
		"Repo\t" +
		"Version\t" +
		"Offset\t" +
		"NumInstances\t" +
		"Owner\t" +
		"Resources\t" +
		"Env"
}

// Tabbed returns the fields of a deployment formatted in a tab delimited list.
func (d *Deployment) Tabbed() string {
	o := "<?>"
	for onr := range d.Owners {
		o = onr
		break
	}

	rs := []string{}
	for k, v := range d.DeployConfig.Resources {
		rs = append(rs, fmt.Sprintf("%s: %s", k, v))
	}
	es := []string{}
	for k, v := range d.DeployConfig.Env {
		es = append(es, fmt.Sprintf("%s: %s", k, v))
	}

	return fmt.Sprintf(
		"%s\t"+ //"Cluster\t" +
			"%s\t"+ //"Repo\t" +
			"%s\t"+ //"Version\t" +
			"%s\t"+ //"Offset\t" +
			"%d\t"+ //"NumInstances\t" +
			"%s\t"+ //"Owner\t" +
			"%s\t"+ //"Resources\t" +
			"%s", //"Env"
		d.ClusterName,
		string(d.SourceID.Repo),
		d.SourceID.Version.String(),
		string(d.SourceID.Dir),
		d.NumInstances,
		o,
		strings.Join(rs, ", "),
		strings.Join(es, ", "),
	)
}

// Name returns the DeployID.
func (d *Deployment) Name() DeployID {
	return DeployID{
		Cluster: d.ClusterName,
		Source:  d.SourceID.Location(),
	}
}

// Equal returns true if two Deployments are equal.
func (d *Deployment) Equal(o *Deployment) bool {
	Log.Vomit.Printf("Comparing: %+ v ?= %+ v", d, o)
	if !(d.ClusterName == o.ClusterName && d.SourceID.Equal(o.SourceID) && d.Kind == o.Kind) { // && len(d.Owners) == len(o.Owners)) {
		Log.Debug.Printf("C: %t V: %t, K: %t, #O: %t", d.ClusterName == o.ClusterName, d.SourceID.Equal(o.SourceID), d.Kind == o.Kind, len(d.Owners) == len(o.Owners))
		return false
	}

	for ownr := range d.Owners {
		if _, has := o.Owners[ownr]; !has {
			return false
		}
	}
	return d.DeployConfig.Equal(o.DeployConfig)
}
