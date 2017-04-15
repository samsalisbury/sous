package sous

import (
	"fmt"
	"sync"
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
		Prior, Post *Deployable
		name        DeploymentID
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
		Log.Info.Printf("Deployment %q has 0 instances, skipping artifact check.", d.ID())
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

// ResolveNames resolves diffs.
func (d *DeployableChans) ResolveNames(r Registry, diff *DeployableChans, errs chan *DiffResolution) {
	d.WaitGroup = sync.WaitGroup{}
	d.Add(4)
	go func() { resolveCreates(r, diff.Start, d.Start, errs); d.Done() }()
	go func() { maybeResolveDeletes(r, diff.Stop, d.Stop, errs); d.Done() }()
	go func() { maybeResolveRetains(r, diff.Stable, d.Stable, errs); d.Done() }()
	go func() { resolvePairs(r, diff.Update, d.Update, errs); d.Done() }()
	go func() { d.Wait(); close(errs) }()
}

func resolveCreates(r Registry, from chan *DeployablePair, to chan *DeployablePair, errs chan *DiffResolution) {
	for dp := range from {
		dep := dp.Post
		Log.Vomit.Printf("Deployment processed, needs artifact: %#v", dep)

		da, err := resolveName(r, dep)
		if err != nil {
			Log.Info.Printf("Unable to create new deployment %q: %s", dep.ID(), err)
			Log.Debug.Printf("Failed create deployment %q: % #v", dep.ID(), dep)
			errs <- err
			continue
		}

		if da.BuildArtifact == nil {
			Log.Info.Printf("Unable to create new deployment %q: no artifact for SourceID %q", dep.ID(), dep.SourceID)
			Log.Debug.Printf("Failed create deployment %q: % #v", dep.ID(), dep)
			continue
		}
		to <- &DeployablePair{name: dp.name, Prior: nil, Post: da}
	}
	close(to)
}

// XXX now that everything is DeployablePairs, this can probably be simplified

func maybeResolveRetains(r Registry, from chan *DeployablePair, to chan *DeployablePair, errs chan *DiffResolution) {
	for dp := range from {
		da := maybeResolveSingle(r, dp.Post)
		to <- &DeployablePair{name: dp.name, Prior: da, Post: da}
	}
	close(to)
}

func maybeResolveDeletes(r Registry, from chan *DeployablePair, to chan *DeployablePair, errs chan *DiffResolution) {
	for dp := range from {
		da := maybeResolveSingle(r, dp.Prior)
		to <- &DeployablePair{name: dp.name, Prior: da, Post: nil}
	}
	close(to)
}

func maybeResolveSingle(r Registry, dep *Deployable) *Deployable {
	Log.Vomit.Printf("Attempting to resolve optional artifact: %#v (stable or deletes don't need images)", dep)
	da, err := resolveName(r, dep)
	if err != nil {
		Log.Debug.Printf("Error resolving stopped or stable deployment (proceeding anyway): %#v: %#v", dep, err)
	}
	return da
}

func resolvePairs(r Registry, from chan *DeployablePair, to chan *DeployablePair, errs chan *DiffResolution) {
	for depPair := range from {
		Log.Vomit.Printf("Pair of deployments processed, needs artifact: %#v", depPair)
		d, err := resolvePair(r, depPair)
		if err != nil {
			Log.Info.Printf("Unable to modify deployment %q: %s", depPair.Post, err)
			Log.Debug.Printf("Failed modify deployment %q: % #v", depPair.ID(), depPair.Post)
			errs <- err
			continue
		}
		if d.Post.BuildArtifact == nil {
			Log.Info.Printf("Unable to modify deployment %q: no artifact for SourceID %q", depPair.ID(), depPair.Post.SourceID)
			Log.Debug.Printf("Failed modify deployment %q: % #v", depPair.ID(), depPair.Post)
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

	return &DeployablePair{name: depPair.name, Prior: prior, Post: post}, err
}
