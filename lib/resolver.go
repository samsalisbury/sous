package sous

import (
	"sync"

	"github.com/opentable/sous/util/firsterr"
)

type (
	// Resolver is responsible for resolving intended and actual deployment
	// states.
	Resolver struct {
		Deployer Deployer
		Registry Registry
	}
)

// NewResolver creates a new Resolver.
func NewResolver(d Deployer, r Registry) *Resolver {
	return &Resolver{
		Deployer: d,
		Registry: r,
	}
}

// Rectify takes a DiffChans and issues the commands to the infrastructure to reconcile the differences
func (r *Resolver) rectify(dcs DiffChans) chan RectificationError {
	d := r.Deployer
	errs := make(chan RectificationError)
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() { d.RectifyCreates(dcs.Created, errs); wg.Done() }()
	go func() { d.RectifyDeletes(dcs.Deleted, errs); wg.Done() }()
	go func() { d.RectifyModifies(dcs.Modified, errs); wg.Done() }()
	go func() { wg.Wait(); close(errs) }()

	return errs
}

// Resolve drives the Sous deployment resolution process. It calls out to the
// appropriate components to compute the intended deployment set, collect the
// actual set, compute the diffs and then issue the commands to rectify those
// differences.
func (r *Resolver) Resolve(intended Deployments, clusters Clusters) error {
	var ads Deployments
	var diffs DiffChans
	var errs chan RectificationError
	return firsterr.Returned(
		func() (e error) { ads, e = r.Deployer.RunningDeployments(clusters); return },
		func() (e error) { return guardImageNamesKnown(r.Registry, intended) },
		func() (e error) { return guardAdvisoriesAcceptable(r.Registry, intended) },
		func() (e error) { diffs = ads.Diff(intended); return nil },
		func() (e error) { errs = r.rectify(diffs); return nil },
		func() (e error) { return foldErrors(errs) },
	)
}

func foldErrors(errs chan RectificationError) error {
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

func guardImageNamesKnown(r Registry, gdm Deployments) error {
	Log.Debug.Print("Collected. Checking readiness to deploy...")
	g := gdm.Snapshot()
	es := make([]error, 0, len(g))
	for _, d := range g {
		_, err := r.GetArtifact(d.SourceID)
		if err != nil {
			es = append(es, err)
		}
	}
	if len(es) > 0 {
		return &MissingImageNamesError{es}
	}
	Log.Debug.Print("Looks good. Proceeding...")
	return nil
}

func guardAdvisoriesAcceptable(r Registry, gdm Deployments) error {
	return nil
}
