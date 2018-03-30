package sous

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

type (
	// A ResolveFilter filters Deployments, DeployStates and Clusters for the
	// purpose of Resolve.resolve().
	ResolveFilter struct {
		//       xxx type-per-field? value?
		Repo     ResolveFieldMatcher
		Offset   ResolveFieldMatcher
		Tag      ResolveFieldMatcher
		Revision ResolveFieldMatcher
		Flavor   ResolveFieldMatcher
		Cluster  ResolveFieldMatcher
		Status   DeployStatus
	}

	// A ResolveFieldMatcher matches against any particular string, or All strings.
	ResolveFieldMatcher struct {
		Match *string
	}
)

// NewResolveFieldMatcher wraps a string in a ResolveFieldMatcher that matches that string.
func NewResolveFieldMatcher(match string) ResolveFieldMatcher {
	return ResolveFieldMatcher{Match: &match}
}

// All returns true if this Matcher matches all values, or false if it matches
// a specific value.
func (matcher ResolveFieldMatcher) All() bool {
	return matcher.Match == nil
}

func (matcher ResolveFieldMatcher) match(against string) bool {
	return matcher.All() || against == *matcher.Match
}

// ValueOr returns the match value for this matcher, or the default value
// provided if the matcher matches all values.
func (matcher ResolveFieldMatcher) ValueOr(def string) string {
	if matcher.All() {
		return def
	}
	return *matcher.Match
}

// Value returns the match value or else returns a descriptive error.
func (matcher ResolveFieldMatcher) Value() (string, error) {
	if matcher.All() {
		return "", fmt.Errorf("field not specified")
	}
	return *matcher.Match, nil
}

func (rf *ResolveFilter) matchRepo(repo string) bool {
	return rf.Repo.match(repo)
}

func (rf *ResolveFilter) matchOffset(offset string) bool {
	return rf.Offset.match(offset)
}

func (rf *ResolveFilter) matchTag(tag string) bool {
	return rf.Tag.match(tag)
}

func (rf *ResolveFilter) matchRevision(rev string) bool {
	return rf.Revision.match(rev)
}

func (rf *ResolveFilter) matchFlavor(flavor string) bool {
	return rf.Flavor.match(flavor)
}

func (rf *ResolveFilter) matchCluster(cluster string) bool {
	return rf.Cluster.match(cluster)
}

func (rf *ResolveFilter) matchDeployStatus(status DeployStatus) bool {
	return (rf.Status == DeployStatusAny || status == rf.Status)
}

// SetTag sets the tag based on a tag string - is ensures the tag parses as semver.
func (rf *ResolveFilter) SetTag(tag string) error {
	tagVersion, err := parseSemverTagWithOptionalPrefix(tag)
	if err != nil {
		return fmt.Errorf("version %q not valid: expected something like [servicename-]1.2.3", tag)
	}

	rf.Tag = NewResolveFieldMatcher(tagVersion.Format(semv.Complete))
	return nil
}

// TagVersion gets the semv parsed from the tag. It returns whatever parsing errors there might be.
func (rf *ResolveFilter) TagVersion() (semv.Version, error) {
	tag, err := rf.Tag.Value()
	if err != nil {
		return semv.Version{}, errors.Wrapf(err, "tag")
	}
	return semv.Parse(tag)
}

// All returns true if the ResolveFilter would allow All deployments.
func (rf *ResolveFilter) All() bool {
	return rf.Repo.All() &&
		rf.Offset.All() &&
		rf.Tag.All() &&
		rf.Revision.All() &&
		rf.Flavor.All() &&
		rf.Cluster.All()
	// xxx && rf.Status.All() ?
}

// SourceLocation returns a SourceLocation and true if this ResolveFilter
// describes a complete specific source location (i.e. it has exact Repo and
// Offset matches set). Otherwise it returns a zero SourceLocation and false.
func (rf *ResolveFilter) SourceLocation() (SourceLocation, bool) {
	if rf.Repo.All() || rf.Offset.All() {
		return SourceLocation{}, false
	}
	return SourceLocation{
		Repo: rf.Repo.ValueOr("<unspecified!>"),
		Dir:  rf.Offset.ValueOr("<unspecified!>"),
	}, true
}

// SourceID returns a SourceID based on the ResolveFilter and a ManifestID
func (rf *ResolveFilter) SourceID(mid ManifestID) (SourceID, error) {
	if rf.Tag.All() {
		return SourceID{}, fmt.Errorf("you must provide the -tag flag")
	}

	newVersion, err := semv.Parse(rf.Tag.ValueOr("<untagged!>"))
	if err != nil {
		return SourceID{}, err
	}

	return mid.Source.SourceID(newVersion), nil
}

// DeploymentID returns a DeploymentID based on the ResolveFilter and a ManifestID
func (rf *ResolveFilter) DeploymentID(mid ManifestID) (DeploymentID, error) {
	if rf.Cluster.All() {
		return DeploymentID{}, fmt.Errorf("you must select a cluster using the -cluster flag")
	}
	return DeploymentID{ManifestID: mid, Cluster: rf.Cluster.ValueOr("<no-cluster!>")}, nil
}

func (rf *ResolveFilter) String() string {

	return fmt.Sprintf(
		"<cluster:%s repo:%s offset:%s flavor:%s tag:%s revision:%s>",
		rf.Cluster.ValueOr("*"),
		rf.Repo.ValueOr("*"),
		rf.Revision.ValueOr("*"),
		rf.Flavor.ValueOr("*"),
		rf.Offset.ValueOr("*"),
		rf.Tag.ValueOr("*"),
	)
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
	return rf.FilterDeployment(&d.Deployment) &&
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
