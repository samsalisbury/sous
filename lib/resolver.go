package sous

import (
	"fmt"
	"sync"

	"github.com/opentable/sous/util/firsterr"
)

type (
	// Resolver is responsible for resolving intended and actual deployment
	// states.
	Resolver struct {
		Deployer Deployer
		Registry Registry
		*ResolveFilter
	}

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

	// DeploymentPredicate takes a *Deployment and returns true if the
	// deployment matches the predicate. Used by Filter to select a subset of a
	// Deployments.
	DeploymentPredicate func(*Deployment) bool
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

// NewResolver creates a new Resolver.
func NewResolver(d Deployer, r Registry, rf *ResolveFilter) *Resolver {
	return &Resolver{
		Deployer:      d,
		Registry:      r,
		ResolveFilter: rf,
	}
}

// Rectify takes a DiffChans and issues the commands to the infrastructure to
// reconcile the differences.
func (r *Resolver) rectify(dcs *DeployableChans, errs chan error) {
	d := r.Deployer
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() { d.RectifyCreates(dcs.Start, errs); wg.Done() }()
	go func() { d.RectifyDeletes(dcs.Stop, errs); wg.Done() }()
	go func() { d.RectifyModifies(dcs.Update, errs); wg.Done() }()
	go func() { wg.Wait(); close(errs) }()
}

// Resolve drives the Sous deployment resolution process. It calls out to the
// appropriate components to compute the intended deployment set, collect the
// actual set, compute the diffs and then issue the commands to rectify those
// differences.
func (r *Resolver) Resolve(intended Deployments, clusters Clusters) error {
	var ads Deployments
	var diffs DiffChans
	var namer *DeployableChans
	errs := make(chan error)
	return firsterr.Set(
		func(*error) { clusters = r.FilteredClusters(clusters) },
		func(e *error) { ads, *e = r.Deployer.RunningDeployments(r.Registry, clusters) },
		func(*error) { intended = intended.Filter(r.FilterDeployment) },
		func(*error) { ads = ads.Filter(r.FilterDeployment) },
		func(*error) { diffs = ads.Diff(intended) },
		func(*error) { namer = NewDeployableChans(10) },
		func(*error) { namer.ResolveNames(r.Registry, &diffs, errs) },
		func(*error) { r.rectify(namer, errs) },
		func(e *error) { *e = foldErrors(errs) },
	)
}

func foldErrors(errs chan error) error {
	re := &ResolveErrors{Causes: []error{}}
	for err := range errs {
		re.Causes = append(re.Causes, err)
		Log.Debug.Printf("resolve error = %+v\n", err)
	}

	if len(re.Causes) > 0 {
		return re
	}
	return nil
}
