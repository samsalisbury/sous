package sous

import (
	"fmt"
	"sync"

	"github.com/opentable/sous/util/firsterr"
	"github.com/pkg/errors"
)

type (
	// Resolver is responsible for resolving intended and actual deployment
	// states.
	Resolver struct {
		Deployer Deployer
		Registry Registry
		*ResolveFilter
	}

	// A ResolveFilter filters Deployments and Clusters for the purpose of Resolve.resolve()
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

// All returns true if the ResolveFilter would allow all deployments
func (rf *ResolveFilter) All() bool {
	return rf.Repo == "" &&
		rf.Offset == "" &&
		rf.Tag == "" &&
		rf.Revision == "" &&
		rf.Flavor == "" &&
		rf.Cluster == ""
}

func (rf *ResolveFilter) String() string {
	return fmt.Sprintf("cluster: %q flavor: %q repo: %q offset: %q tag: %q revision %q",
		rf.Cluster, rf.Flavor, rf.Repo, rf.Offset, rf.Tag, rf.Revision)
}

// FilteredClusters returns a new Clusters relevant to the Deployments that this ResolveFilter would permit
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

// FilterDeployment behaves as a DeploymentPredicate, filtering Deployments if they match its criteria
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
// TODO: @nyarly can you provide a description of what this function does
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

// Rectify takes a DiffChans and issues the commands to the infrastructure to reconcile the differences
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
	return firsterr.Returned(
		func() (e error) { clusters = r.FilteredClusters(clusters); return },
		func() (e error) { ads, e = r.Deployer.RunningDeployments(r.Registry, clusters); return },
		func() (e error) { intended = intended.Filter(r.FilterDeployment); return },
		func() (e error) { ads = ads.Filter(r.FilterDeployment); return },
		func() (e error) { return GuardImages(r.Registry, intended) },
		func() (e error) { diffs = ads.Diff(intended); return },
		func() (e error) { namer = NewDeployableChans(10); return },
		func() (e error) { namer.ResolveNames(r.Registry, &diffs, errs); return },
		func() (e error) { r.rectify(namer, errs); return },
		func() (e error) { return foldErrors(errs) },
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

// GuardImage checks that a deployment is valid before deploying it
func GuardImage(r Registry, d *Deployment) error {
	if d.NumInstances == 0 { // we're not deploying any of these, so it can be wrong for the moment
		return nil
	}
	art, err := r.GetArtifact(d.SourceID)
	if err != nil {
		return &MissingImageNameError{err}
	}
	for _, q := range art.Qualities {
		if q.Kind == `advisory` {
			if q.Name == "" {
				return nil
			}
			advisoryIsValid := false
			var allowedAdvisories []string
			if d.Cluster == nil {
				return fmt.Errorf("nil cluster on deployment %q", d)
			}
			allowedAdvisories = d.Cluster.AllowedAdvisories
			for _, aa := range allowedAdvisories {
				if aa == q.Name {
					advisoryIsValid = true
					break
				}
			}
			if !advisoryIsValid {
				return &UnacceptableAdvisory{q, &d.SourceID}
			}
		}
	}
	return nil
}

// GuardImages checks that all deployments have valid artifacts ready to deploy.
func GuardImages(r Registry, gdm Deployments) error {
	Log.Debug.Print("Collected. Checking readiness to deploy...")
	g := gdm.Snapshot()
	es := make([]error, 0, len(g))
	for _, d := range g {
		err := GuardImage(r, d)
		if err != nil {
			es = append(es, err)
		}
	}
	if len(es) > 0 {
		return errors.Wrap(&ResolveErrors{es}, "guard")
	}
	Log.Debug.Print("Looks good. Proceeding...")
	return nil
}
