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
		*ResolveFilter
		ls logging.LogSink
	}

	// DeploymentPredicate takes a *Deployment and returns true if the
	// deployment matches the predicate. Used by Filter to select a subset of a
	// Deployments.
	DeploymentPredicate func(*Deployment) bool
)

// NewResolver creates a new Resolver.
func NewResolver(d Deployer, r Registry, rf *ResolveFilter, ls logging.LogSink) *Resolver {
	return &Resolver{
		Deployer:      d,
		Registry:      r,
		ResolveFilter: rf,
		ls:            ls,
	}
}

var globalQueueSet *R11nQueueSet

// queueDiffs adds a rectification for each required change in DeployableChans,
// as long as there is no planned or currently executing resolution for the
// DeploymentID relating to that rectification.
func (r *Resolver) queueDiffs(dcs *DeployableChans, results chan DiffResolution) {
	if globalQueueSet == nil {
		globalQueueSet = NewR11nQueueSet(R11nQueueStartWithHandler(
			func(qr *QueuedR11n) DiffResolution {
				qr.Rectification.Begin(r.Deployer)
				return qr.Rectification.Wait()
			}))
	}

	var wg sync.WaitGroup
	for p := range dcs.Pairs {
		sr := NewRectification(*p)
		queued, ok := globalQueueSet.PushIfEmpty(sr)
		if !ok {
			reportR11nAnomaly(r.ls, sr, r11nDroppedQueueNotEmpty)
			continue
		}
		wg.Add(1)
		did := p.ID() // Capture did from the range var p outside the goroutine.
		go func() {
			defer wg.Done()
			result, ok := globalQueueSet.Wait(did, queued.ID)
			if !ok {
				reportR11nAnomaly(r.ls, sr, r11nWentMissing)
			}
			results <- result
		}()
	}
	wg.Wait()
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
	intended = intended.Filter(r.FilterDeployment)

	return NewResolveRecorder(intended, func(recorder *ResolveRecorder) {
		var actual DeployStates
		var diffs *DeployableChans
		var logger *DeployableChans
		ctx := context.Background()

		recorder.performPhase("filtering clusters", func() error {
			clusters = r.FilteredClusters(clusters)
			return nil
		})

		recorder.performPhase("getting running deployments", func() error {
			var err error
			actual, err = r.Deployer.RunningDeployments(r.Registry, clusters)
			return err
		})

		recorder.performPhase("filtering running deployments", func() error {
			actual = actual.Filter(r.FilterDeployStates)
			return nil
		})

		recorder.performPhase("generating diff", func() error {
			diffs = actual.Diff(intended)
			return nil
		})

		recorder.performPhase("resolving deployment artifacts", func() error {
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
			return nil
		})

		recorder.performPhase("rectification", func() error {
			r.queueDiffs(logger, recorder.Log)
			return nil
		})

		logger.Wait()
	})
}
