package sous

import "fmt"

type (
	// A ResolveFilter filters Deployments, DeployStates and Clusters for the
	// purpose of Resolve.resolve().
	ResolveFilter struct {
		Repo     string
		Offset   ResolveFieldMatcher
		Tag      string
		Revision string
		Flavor   ResolveFieldMatcher
		Cluster  string
		Status   DeployStatus
	}

	// A ResolveFieldMatcher matches against any particular string, or all strings.
	ResolveFieldMatcher struct {
		All   bool
		Match string
	}
)

func (matcher ResolveFieldMatcher) match(against string) bool {
	return matcher.All || against == matcher.Match
}

func (rf *ResolveFilter) matchRepo(repo string) bool {
	return rf.Repo == "" || repo == rf.Repo
}

func (rf *ResolveFilter) matchOffset(offset string) bool {
	return rf.Offset.match(offset)
}

func (rf *ResolveFilter) matchTag(tag string) bool {
	return rf.Tag == "" || tag == rf.Tag
}

func (rf *ResolveFilter) matchRevision(rev string) bool {
	return rf.Revision == "" || rev == rf.Revision
}

func (rf *ResolveFilter) matchFlavor(flavor string) bool {
	return rf.Flavor.match(flavor)
}

func (rf *ResolveFilter) matchCluster(cluster string) bool {
	return rf.Cluster == "" || cluster == rf.Cluster
}

func (rf *ResolveFilter) matchDeployStatus(status DeployStatus) bool {
	return (rf.Status == DeployStatusAny || status == rf.Status)
}

// All returns true if the ResolveFilter would allow all deployments.
func (rf *ResolveFilter) All() bool {
	return rf.Repo == "" &&
		rf.Offset.All &&
		rf.Tag == "" &&
		rf.Revision == "" &&
		rf.Flavor.All &&
		rf.Cluster == ""
}

func (rf *ResolveFilter) String() string {
	cl, fl, rp, of, tg, rv := rf.Cluster, rf.Flavor.Match, rf.Repo, rf.Offset.Match, rf.Tag, rf.Revision
	if cl == "" {
		cl = `*`
	}
	if rf.Flavor.All {
		fl = `*`
	}
	if rp == "" {
		rp = `*`
	}
	if rf.Offset.All {
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
		if !rf.FilterClusterName(n) {
			continue
		}
		newC[n] = c // c is a *Cluster, so be aware they need to not be changed
	}
	return newC
}

// FilterClusterName returns true if the given string would be matched by this
// ResolveFilter as a ClusterName.
func (rf *ResolveFilter) FilterClusterName(name string) bool {
	return rf.matchCluster(name)
}

// FilterDeployment behaves as a DeploymentPredicate, filtering Deployments if
// they match its criteria.
func (rf *ResolveFilter) FilterDeployment(d *Deployment) bool {
	return rf.matchRepo(d.SourceID.Location.Repo) &&
		rf.matchOffset(d.SourceID.Location.Dir) &&
		rf.matchTag(d.SourceID.Version.String()) &&
		rf.matchRevision(d.SourceID.RevID()) &&
		rf.matchFlavor(d.Flavor) &&
		rf.matchCluster(d.ClusterName)
}

// FilterDeployStates is similar to FilterDeployment, but also filters by
// DeployStatus.
func (rf *ResolveFilter) FilterDeployStates(d *DeployState) bool {
	return rf.FilterDeployment(d.Deployment) &&
		rf.matchDeployStatus(d.Status)
}

// FilterManifest returns true if the Manifest is matched by this ResolveFilter.
func (rf *ResolveFilter) FilterManifest(m *Manifest) bool {
	return rf.FilterManifestID(m.ID())
}

// FilterManifestID returns true if the ManifestID is matched by this ResolveFilter.
func (rf *ResolveFilter) FilterManifestID(m ManifestID) bool {
	return rf.matchRepo(m.Source.Repo) &&
		rf.matchOffset(m.Source.Dir) &&
		rf.matchFlavor(m.Flavor)
}
