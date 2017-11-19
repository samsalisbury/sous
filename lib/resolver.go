package sous

import (
	"context"
	"sync"

	"github.com/opentable/sous/util/logging"
)

type (
	// Resolver is responsible for resolving intended and actual deployment
	// states.
	Resolver struct {
		Deployer Deployer
		Registry Registry
		Filter   *ResolveFilter
		ls       logging.LogSink
	}

	// DeploymentPredicate takes a *Deployment and returns true if the
	// deployment matches the predicate. Used by Filter to select a subset of a
	// Deployments.
	DeploymentPredicate func(*Deployment) bool
)

// NewResolver creates a new Resolver.
func NewResolver(d Deployer, r Registry, rf *ResolveFilter, ls logging.LogSink) *Resolver {
	return &Resolver{
		Deployer: d,
		Registry: r,
		Filter:   rf,
		ls:       ls,
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

func (r *Resolver) reportStable(stable <-chan *DeployablePair, results chan<- DiffResolution) {
	for dp := range stable {
		dep := dp.Prior
		rez := DiffResolution{
			DeploymentID: dep.ID(),
			Desc:         StableDiff,
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
	intended = intended.Filter(r.Filter.FilterDeployment)

	return NewResolveRecorder(intended, func(recorder *ResolveRecorder) {
		var actual DeployStates
		var diffs *DeployableChans
		var logger *DeployableChans

		recorder.performGuaranteedPhase("filtering clusters", func() {
			clusters = r.Filter.FilteredClusters(clusters)
		})

		recorder.performPhase("getting running deployments", func() error {
			var err error
			actual, err = r.Deployer.RunningDeployments(r.Registry, clusters)
			return err
		})

		recorder.performGuaranteedPhase("filtering running deployments", func() {
			actual = actual.Filter(r.Filter.FilterDeployStates)
		})

		recorder.performGuaranteedPhase("generating diff", func() {
			diffs = actual.Diff(intended)
		})

		ctx := context.Background()
		recorder.performGuaranteedPhase("resolving deployment artifacts", func() {
			namer := diffs.ResolveNames(ctx, r.Registry)
			logger = namer.Log(ctx, r.ls)
			logger.Add(1)
			go func() {
				for err := range logger.Errs {
					recorder.Log <- *err
					//DiffResolution{Error: &ErrorWrapper{error: err}}
				}
				logger.Done()
			}()
			// TODO: ResolveNames should take rs.Log instead of errs.
		})

		recorder.performGuaranteedPhase("rectification", func() {
			r.rectify(logger, recorder.Log)
		})
		logger.Wait()
	})
}
