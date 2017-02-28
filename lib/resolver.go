package sous

import "sync"

type (
	// Resolver is responsible for resolving intended and actual deployment
	// states.
	Resolver struct {
		Deployer Deployer
		Registry Registry
		*ResolveFilter
	}

	// DeploymentPredicate takes a *Deployment and returns true if the
	// deployment matches the predicate. Used by Filter to select a subset of a
	// Deployments.
	DeploymentPredicate func(*Deployment) bool
)

// NewResolver creates a new Resolver.
func NewResolver(d Deployer, r Registry, rf *ResolveFilter) *Resolver {
	return &Resolver{
		Deployer:      d,
		Registry:      r,
		ResolveFilter: rf,
	}
}

// rectify takes a DiffChans and issues the commands to the infrastructure to
// reconcile the differences.
func (r *Resolver) rectify(dcs *DeployableChans, results chan DiffResolution) {
	d := r.Deployer
	wg := &sync.WaitGroup{}
	wg.Add(4)
	go func() { d.RectifyCreates(dcs.Start, results); wg.Done() }()
	go func() { d.RectifyDeletes(dcs.Stop, results); wg.Done() }()
	go func() { d.RectifyModifies(dcs.Update, results); wg.Done() }()
	go func() { r.reportStable(dcs.Stable, results); wg.Done() }()
	wg.Wait()
}

func (r *Resolver) reportStable(stable <-chan *Deployable, results chan<- DiffResolution) {
	for dep := range stable {
		rez := DiffResolution{
			DeployID: dep.ID(),
			Desc:     StableDiff,
		}
		if dep.Status == DeployStatusPending {
			rez.Desc = ComingDiff
		}
		if dep.Status == DeployStatusFailed {
			rez.Error = WrapResolveError(&FailedStatusError{})
		}
		results <- rez
	}
}

// Begin is similar to Resolve, except that it returns a ResolveRecorder almost
// immediately, which can be queried for information about the ongoing
// resolution. You can check if resolution is finished by calling Done() on the
// returned ResolveRecorder.
//
// This process drives the Sous deployment resolution process. It calls out to
// the appropriate components to compute the intended deployment set, collect
// the actual set, compute the diffs and then issue the commands to rectify
// those differences.
func (r *Resolver) Begin(intended Deployments, clusters Clusters) *ResolveRecorder {
	return NewResolveRecorder(func(recorder *ResolveRecorder) {
		recorder.performGuaranteedPhase("filtering clusters", func() {
			clusters = r.FilteredClusters(clusters)
		})

		recorder.performGuaranteedPhase("filtering intended deployments", func() {
			intended = intended.Filter(r.FilterDeployment)
		})

		var actual DeployStates

		recorder.performPhase("getting running deployments", func() error {
			var err error
			actual, err = r.Deployer.RunningDeployments(r.Registry, clusters)
			return err
		})

		recorder.performGuaranteedPhase("filtering running deployments", func() {
			actual = actual.Filter(r.FilterDeployStates)
		})

		var diffs DiffChans
		recorder.performGuaranteedPhase("generating diff", func() {
			diffs = actual.IgnoringStatus().Diff(intended)
		})

		//recorder.TasksStarting = actual.Filter(func(ds *DeployState) bool {
		//	ds.Status = DeployStatusPending
		//})

		namer := NewDeployableChans(10)
		var wg sync.WaitGroup
		recorder.performGuaranteedPhase("resolving deployment artifacts", func() {
			errs := make(chan error)
			wg.Add(1)
			go func() {
				for err := range errs {
					recorder.Log <- DiffResolution{Error: &ErrorWrapper{error: err}}
				}
				wg.Done()
			}()
			// TODO: ResolveNames should take rs.Log instead of errs.
			namer.ResolveNames(r.Registry, &diffs, errs)
		})

		recorder.performGuaranteedPhase("rectification", func() {
			r.rectify(namer, recorder.Log)
		})
		wg.Wait()
	})
}
