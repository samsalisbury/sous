package sous

import (
	"context"
	"fmt"
	"sync"

	"github.com/opentable/sous/util/logging"
)

type (
	// A DeployableChans is a bundle of channels describing actions to take on a
	// cluster
	DeployableChans struct {
		Start, Stop, Stable, Update chan *DeployablePair
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
	}
}

// Close closes all the channels in a DeployableChans in a single action
func (d *DeployableChans) Close() {
	close(d.Start)
	close(d.Stop)
	close(d.Stable)
	close(d.Update)
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
	Start(dp *DeployablePair) *DeployablePair
	Stop(dp *DeployablePair) *DeployablePair
	Stable(dp *DeployablePair) *DeployablePair
	Update(dp *DeployablePair) *DeployablePair
	StartClosed()
	StopClosed()
	StableClosed()
	UpdateClosed()
}

// Pipeline attaches a DeployableProcessor to the DeployableChans, and returns a new DeployableChans.
func (d *DeployableChans) Pipeline(ctx context.Context, proc DeployableProcessor) *DeployableChans {
	out := NewDeployableChans(1)

	process := func(from, to chan *DeployablePair, doProc func(dp *DeployablePair) *DeployablePair, closed func()) {
		defer closed()
		defer close(to)
		for {
			select {
			case <-ctx.Done():
				return
			case dp, ok := <-from:
				if ok {
					if proked := doProc(dp); proked != nil {
						to <- proked
					}
				} else {
					return
				}
			}
		}
	}

	go process(d.Start, out.Start, proc.Start, proc.StartClosed)
	go process(d.Stop, out.Stop, proc.Stop, proc.StopClosed)
	go process(d.Stable, out.Stable, proc.Stable, proc.StableClosed)
	go process(d.Update, out.Update, proc.Update, proc.UpdateClosed)
	return out
}

// DeployablePassThrough implements DeployableProcessor trivially: each method simply passes its argument on
type DeployablePassThrough struct{}

func (DeployablePassThrough) Start(dp *DeployablePair) *DeployablePair {
	return dp
}

func (DeployablePassThrough) Stop(dp *DeployablePair) *DeployablePair {
	return dp
}

func (DeployablePassThrough) Stable(dp *DeployablePair) *DeployablePair {
	return dp
}

func (DeployablePassThrough) Update(dp *DeployablePair) *DeployablePair {
	return dp
}

func (DeployablePassThrough) StartClosed() {}

func (DeployablePassThrough) StopClosed() {}

func (DeployablePassThrough) StableClosed() {}

func (DeployablePassThrough) UpdateClosed() {}

type nameResolver struct {
	DeployablePassThrough
	wait     sync.WaitGroup
	registry Registry
	errs     chan *DiffResolution
}

func (names *nameResolver) Start(dp *DeployablePair) *DeployablePair {
	dep := dp.Post
	logging.Log.Vomit.Printf("Deployment processed, needs artifact: %#v", dep)

	da, err := resolveName(names.registry, dep)
	if err != nil {
		logging.Log.Info.Printf("Unable to create new deployment %q: %s", dep.ID(), err)
		logging.Log.Debug.Printf("Failed create deployment %q: % #v", dep.ID(), dep)
		names.errs <- err
		return nil
	}

	if da.BuildArtifact == nil {
		logging.Log.Info.Printf("Unable to create new deployment %q: no artifact for SourceID %q", dep.ID(), dep.SourceID)
		logging.Log.Debug.Printf("Failed create deployment %q: % #v", dep.ID(), dep)
		return nil
	}
	return &DeployablePair{ExecutorData: dp.ExecutorData, name: dp.name, Prior: nil, Post: da}
}

func (names *nameResolver) Stop(dp *DeployablePair) *DeployablePair {
	da := maybeResolveSingle(names.registry, dp.Prior)
	return &DeployablePair{ExecutorData: dp.ExecutorData, name: dp.name, Prior: da, Post: nil}
}

func (names *nameResolver) Stable(dp *DeployablePair) *DeployablePair {
	da := maybeResolveSingle(names.registry, dp.Post)
	return &DeployablePair{ExecutorData: dp.ExecutorData, name: dp.name, Prior: da, Post: da}
}

func (names *nameResolver) StartClosed() {
	names.wait.Done()
}

func (names *nameResolver) StopClosed() {
	names.wait.Done()
}

func (names *nameResolver) StableClosed() {
	names.wait.Done()
}

func (names *nameResolver) UpdateClosed() {
	names.wait.Done()
}

func (names *nameResolver) ready() {
	names.wait.Add(4)
	go func() {
		names.wait.Wait()
		close(names.errs)
	}()
}

// ResolveNames resolves diffs.
func (d *DeployableChans) ResolveNames(r Registry, diff *DeployableChans, errs chan *DiffResolution) {
	names := &nameResolver{
		registry: r,
		errs:     errs,
	}
	names.ready()

	//out :=
	d.Pipeline(context.Background(), names)
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

func resolvePairs(r Registry, from chan *DeployablePair, to chan *DeployablePair, errs chan *DiffResolution) {
	for depPair := range from {
		logging.Log.Vomit.Printf("Pair of deployments processed, needs artifact: %#v", depPair)
		d, err := resolvePair(r, depPair)
		if err != nil {
			logging.Log.Info.Printf("Unable to modify deployment %q: %s", depPair.Post, err)
			logging.Log.Debug.Printf("Failed modify deployment %q: % #v", depPair.ID(), depPair.Post)
			errs <- err
			continue
		}
		if d.Post.BuildArtifact == nil {
			logging.Log.Info.Printf("Unable to modify deployment %q: no artifact for SourceID %q", depPair.ID(), depPair.Post.SourceID)
			logging.Log.Debug.Printf("Failed modify deployment %q: % #v", depPair.ID(), depPair.Post)
			continue
		}
		to <- d
	}
	close(to)
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
