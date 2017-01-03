package sous

import "fmt"

type (
	// A ResolveFilter filters Deployments and Clusters for the purpose of
	// Resolve.resolve().
	ResolveFilter struct {
		Repo     string
		Offset   string
		Tag      string
		Revision string
		Flavor   string
		Cluster  string
	}
)

// All returns true if the ResolveFilter would allow all deployments.
func (rf *ResolveFilter) All() bool {
	return rf.Repo == "" &&
		rf.Offset == "" &&
		rf.Tag == "" &&
		rf.Revision == "" &&
		rf.Flavor == "" &&
		rf.Cluster == ""
}

func (rf *ResolveFilter) String() string {
	cl, fl, rp, of, tg, rv := rf.Cluster, rf.Flavor, rf.Repo, rf.Offset, rf.Tag, rf.Revision
	if cl == "" {
		cl = `*`
	}
	if fl == "" {
		fl = `*`
	}
	if rp == "" {
		rp = `*`
	}
	if of == "" {
		of = `*`
	}
	if tg == "" {
		tg = `*`
	}
	if rv == "" {
		rv = `*`
	}
	return fmt.Sprintf("<cluster:%s repo:%s offset:%s flavor:%s tag:%s revision:%s>",
		cl, rp, of, fl, tg, rv)
}

// FilteredClusters returns a new Clusters relevant to the Deployments that this
// ResolveFilter would permit.
func (rf *ResolveFilter) FilteredClusters(c Clusters) Clusters {
	newC := make(Clusters)
	for n, c := range c {
		if rf.Cluster != "" && n != rf.Cluster {
			continue
		}
		newC[n] = c // c is a *Cluster, so be aware they need to not be changed
	}
	return newC
}

// FilterDeployment behaves as a DeploymentPredicate, filtering Deployments if
// they match its criteria.
func (rf *ResolveFilter) FilterDeployment(d *Deployment) bool {
	if rf.Repo != "" && d.SourceID.Location.Repo != rf.Repo {
		return false
	}
	if rf.Offset != "" && d.SourceID.Location.Dir != rf.Offset {
		return false
	}
	if rf.Tag != "" && d.SourceID.Version.String() != rf.Tag {
		return false
	}
	if rf.Revision != "" && d.SourceID.RevID() != rf.Revision {
		return false
	}
	if rf.Flavor != "" && d.Flavor != rf.Flavor {
		return false
	}
	if rf.Cluster != "" && d.ClusterName != rf.Cluster {
		return false
	}
	return true
}

// FilterManifest returns true if ???
// TODO: @nyarly can you provide a description of what this function does?
func (rf *ResolveFilter) FilterManifest(m *Manifest) bool {
	if rf.Repo != "" && m.Source.Repo != rf.Repo {
		return false
	}
	if rf.Offset != "" && m.Source.Dir != rf.Offset {
		return false
	}
	return true
}
