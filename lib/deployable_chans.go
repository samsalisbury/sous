package sous

import (
	"context"
	"fmt"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

type (
	// A DeployableChans is a bundle of channels describing actions to take on a
	// cluster
	DeployableChans struct {
		Start, Stop, Stable, Update chan *DeployablePair
		Errs                        chan *DiffResolution
		sync.WaitGroup
	}

	// A DeployablePair is a pair of deployables, describing a "before and after"
	// situation, where the Prior Deployable is the known state and the Post
	// Deployable is the desired state.
	DeployablePair struct {
		Prior, Post  *Deployable
		name         DeploymentID
		ExecutorData interface{}
	}

	diffSet struct {
		New, Gone, Same, Changed DeployablePairs
	}
)

// NewDeployableChans returns a new DeployableChans with channels buffered to
// size.
func NewDeployableChans(size ...int) *DeployableChans {
	var s int
	if len(size) > 0 {
		s = size[0]
	}
	return &DeployableChans{
		Start:  make(chan *DeployablePair, s),
		Stop:   make(chan *DeployablePair, s),
		Stable: make(chan *DeployablePair, s),
		Update: make(chan *DeployablePair, s),
		Errs:   make(chan *DiffResolution, s),
	}
}

// Close closes all the channels in a DeployableChans in a single action
func (d *DeployableChans) Close() {
	close(d.Start)
	close(d.Stop)
	close(d.Stable)
	close(d.Update)
	close(d.Errs)
}

func (d *DeployableChans) collect() diffSet {
	ds := newDiffSet()

	for n := range d.Start {
		ds.New = append(ds.New, n)
	}
	for g := range d.Stop {
		ds.Gone = append(ds.Gone, g)
	}
	for s := range d.Stable {
		ds.Same = append(ds.Same, s)
	}
	for m := range d.Update {
		ds.Changed = append(ds.Changed, m)
	}
	return ds
}

// GuardImage checks that a deployment is valid before deploying it.
func GuardImage(r Registry, d *Deployment) (*BuildArtifact, error) {
	if d.NumInstances == 0 {
		logging.Log.Info.Printf("Deployment %q has 0 instances, skipping artifact check.", d.ID())
		return nil, nil
	}
	art, err := r.GetArtifact(d.SourceID)
	if err != nil {
		return nil, &MissingImageNameError{err}
	}
	for _, q := range art.Qualities {
		if q.Kind == "advisory" {
			if q.Name == "" {
				continue
			}
			advisoryIsValid := false
			var allowedAdvisories []string
			if d.Cluster == nil {
				return nil, fmt.Errorf("nil cluster on deployment %q", d)
			}
			allowedAdvisories = d.Cluster.AllowedAdvisories
			for _, aa := range allowedAdvisories {
				if aa == q.Name {
					advisoryIsValid = true
					break
				}
			}
			if !advisoryIsValid {
				return nil, &UnacceptableAdvisory{q, &d.SourceID}
			}
		}
	}
	return art, err
}

// ID returns the ID of this DeployablePair.
func (dp *DeployablePair) ID() DeploymentID {
	return dp.name
}

// DeployableProcessor processes DeployablePairs off of a DeployableChans channel
type DeployableProcessor interface {
	Start(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Stop(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Stable(dp *DeployablePair) (*DeployablePair, *DiffResolution)
	Update(dp *DeployablePair) (*DeployablePair, *DiffResolution)
}

// Pipeline attaches a DeployableProcessor to the DeployableChans, and returns a new DeployableChans.
func (d *DeployableChans) Pipeline(ctx context.Context, proc DeployableProcessor) *DeployableChans {
	out := NewDeployableChans(1)

	wg := sync.WaitGroup{}
	wg.Add(4)

	go func() {
		for rez := range d.Errs {
			out.Errs <- rez
		}
		wg.Wait()
		close(out.Errs)
	}()

	process := func(from, to chan *DeployablePair, doProc func(dp *DeployablePair) (*DeployablePair, *DiffResolution)) {
		defer close(to)
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case dp, ok := <-from:
				if ok {
					proked, rez := doProc(dp)
					if rez != nil {
						out.Errs <- rez
					}
					if proked != nil {
						to <- proked
					}
				} else {
					return
				}
			}
		}
	}

	go process(d.Start, out.Start, proc.Start)
	go process(d.Stop, out.Stop, proc.Stop)
	go process(d.Stable, out.Stable, proc.Stable)
	go process(d.Update, out.Update, proc.Update)
	return out
}

// DeployablePassThrough implements DeployableProcessor trivially: each method simply passes its argument on
type DeployablePassThrough struct{}

// Start returns its argument
func (DeployablePassThrough) Start(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	return dp, nil
}

// Stop returns its argument
func (DeployablePassThrough) Stop(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	return dp, nil
}

// Stable returns its argument
func (DeployablePassThrough) Stable(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	return dp, nil
}

// Update returns its argument
func (DeployablePassThrough) Update(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	return dp, nil
}

type nameResolver struct {
	registry Registry
}

func (names *nameResolver) Start(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	spew.Dump(names)
	dep := dp.Post
	logging.Log.Vomit.Printf("Deployment processed, needs artifact: %#v", dep)

	da, err := resolveName(names.registry, dep)
	if err != nil {
		logging.Log.Info.Printf("Unable to create new deployment %q: %s", dep.ID(), err)
		logging.Log.Debug.Printf("Failed create deployment %q: % #v", dep.ID(), dep)
		return nil, err
	}

	if da.BuildArtifact == nil {
		logging.Log.Info.Printf("Unable to create new deployment %q: no artifact for SourceID %q", dep.ID(), dep.SourceID)
		logging.Log.Debug.Printf("Failed create deployment %q: % #v", dep.ID(), dep)
		return nil, &DiffResolution{
			DeploymentID: dp.ID(),
			Desc:         "not created",
			Error:        WrapResolveError(errors.Errorf("Unable to create new deployment %q: no artifact for SourceID %q", dep.ID(), dep.SourceID)),
		}
	}
	return &DeployablePair{ExecutorData: dp.ExecutorData, name: dp.name, Prior: nil, Post: da}, nil
}

func (names *nameResolver) Update(depPair *DeployablePair) (*DeployablePair, *DiffResolution) {
	logging.Log.Vomit.Printf("Pair of deployments processed, needs artifact: %#v", depPair)
	d, err := resolvePair(names.registry, depPair)
	if err != nil {
		logging.Log.Info.Printf("Unable to modify deployment %q: %s", depPair.Post, err)
		logging.Log.Debug.Printf("Failed modify deployment %q: % #v", depPair.ID(), depPair.Post)
		return nil, err
	}
	if d.Post.BuildArtifact == nil {
		logging.Log.Info.Printf("Unable to modify deployment %q: no artifact for SourceID %q", depPair.ID(), depPair.Post.SourceID)
		logging.Log.Debug.Printf("Failed modify deployment %q: % #v", depPair.ID(), depPair.Post)
		return nil, &DiffResolution{
			DeploymentID: depPair.ID(),
			Desc:         "not updated",
			Error:        WrapResolveError(errors.Errorf("Unable to modify new deployment %q: no artifact for SourceID %q", depPair.ID(), depPair.Post.SourceID)),
		}
	}
	return d, nil
}

// Stop always returns no error because we don't need a deploy artifact to delete a deploy
func (names *nameResolver) Stop(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	da := maybeResolveSingle(names.registry, dp.Prior)
	return &DeployablePair{ExecutorData: dp.ExecutorData, name: dp.name, Prior: da, Post: nil}, nil
}

// Stable always returns no error because we don't need a deploy artifact for unchanged deploys
func (names *nameResolver) Stable(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	da := maybeResolveSingle(names.registry, dp.Post)
	return &DeployablePair{ExecutorData: dp.ExecutorData, name: dp.name, Prior: da, Post: da}, nil
}

// ResolveNames resolves diffs.
func (d *DeployableChans) ResolveNames(r Registry) *DeployableChans {
	names := &nameResolver{registry: r}

	return d.Pipeline(context.Background(), names)
}

// XXX now that everything is DeployablePairs, this can probably be simplified

func maybeResolveSingle(r Registry, dep *Deployable) *Deployable {
	logging.Log.Vomit.Printf("Attempting to resolve optional artifact: %#v (stable or deletes don't need images)", dep)
	da, err := resolveName(r, dep)
	if err != nil {
		logging.Log.Debug.Printf("Error resolving stopped or stable deployment (proceeding anyway): %#v: %#v", dep, err)
	}
	return da
}

func resolveName(r Registry, d *Deployable) (*Deployable, *DiffResolution) {
	art, err := GuardImage(r, d.Deployment)
	if err != nil {
		return d, &DiffResolution{
			DeploymentID: d.ID(),
			Error:        &ErrorWrapper{error: err},
		}
	}
	d.BuildArtifact = art
	return d, nil
}

func resolvePair(r Registry, depPair *DeployablePair) (*DeployablePair, *DiffResolution) {
	prior, _ := resolveName(r, depPair.Prior)
	post, err := resolveName(r, depPair.Post)

	return &DeployablePair{ExecutorData: depPair.ExecutorData, name: depPair.name, Prior: prior, Post: post}, err
}
