package sous

import (
	"fmt"
	"sync"
)

type (
	// A DeployableChans is a bundle of channels describing actions to take on a
	// cluster
	DeployableChans struct {
		Start, Stop, Stable chan *Deployable
		Update              chan *DeployablePair
		sync.WaitGroup
	}

	// A DeployablePair is a pair of deployables, describing a "before and after"
	// situation, where the Prior Deployable is the known state and the Post
	// Deployable is the desired state.
	DeployablePair struct {
		Prior, Post *Deployable
		name        DeployID
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
		Start:  make(chan *Deployable, s),
		Stop:   make(chan *Deployable, s),
		Stable: make(chan *Deployable, s),
		Update: make(chan *DeployablePair, s),
	}
}

// GuardImage checks that a deployment is valid before deploying it
func GuardImage(r Registry, d *Deployment) (art *BuildArtifact, err error) {
	if d.NumInstances == 0 { // we're not deploying any of these, so it can be wrong for the moment
		return
	}
	art, err = r.GetArtifact(d.SourceID)
	if err != nil {
		return nil, &MissingImageNameError{err}
	}
	for _, q := range art.Qualities {
		if q.Kind == `advisory` {
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
	return
}

// ID returns the ID of this DeployablePair.
func (dp *DeployablePair) ID() DeployID {
	return dp.name
}

// ResolveNames resolves diffs.
func (dc *DeployableChans) ResolveNames(r Registry, diff *DiffChans, errs chan error) {
	dc.WaitGroup = sync.WaitGroup{}
	dc.Add(4)
	go func() { resolveSingles(r, diff.Created, dc.Start, errs); dc.Done() }()
	go func() { unresolvedSingles(r, diff.Deleted, dc.Stop, errs); dc.Done() }()
	go func() { unresolvedSingles(r, diff.Retained, dc.Stable, errs); dc.Done() }()
	go func() { resolvePairs(r, diff.Modified, dc.Update, errs); dc.Done() }()
	go func() { dc.Wait(); close(errs) }()
}

func unresolvedSingles(r Registry, from chan *Deployment, to chan *Deployable, errs chan error) {
	for dep := range from {
		Log.Vomit.Printf("Deployment processed w/o needing artifact: %#v", dep)
		da, err := resolveName(r, dep)
		if err != nil {
			Log.Debug.Printf("Error resolving stopped or stable deployment (proceeding anyway): %#v: %#v", dep, err)
		}
		to <- da
	}
	close(to)
}

func resolveSingles(r Registry, from chan *Deployment, to chan *Deployable, errs chan error) {
	for dep := range from {
		Log.Vomit.Printf("Deployment processed, needs artifact: %#v", dep)

		da, err := resolveName(r, dep)
		if err != nil {
			Log.Debug.Printf("Error resolving deployment (won't deploy): %#v: %#v", dep, err)
			errs <- err
			continue
		}
		if da.BuildArtifact == nil {
			Log.Debug.Printf("No artifact known for created deployment (won't deploy): %#v", dep)
			continue
		}
		to <- da
	}
	close(to)
}

func resolvePairs(r Registry, from chan *DeploymentPair, to chan *DeployablePair, errs chan error) {
	for depPair := range from {
		Log.Vomit.Printf("Pair of deployments processed, needs artifact: %#v", depPair)
		d, err := resolvePair(r, depPair)
		if err != nil {
			Log.Debug.Printf("Error resolving post deployment of change pair (won't deploy): %#v: %#v", depPair.Post, err)
			errs <- err
			continue
		}
		if d.Post.BuildArtifact == nil {
			Log.Debug.Printf("No artifact known for post deployment in change pair (won't deploy): %#v", depPair.Post)
			continue
		}
		to <- d
	}
	close(to)
}

func resolveName(r Registry, dep *Deployment) (d *Deployable, err error) {
	d = &Deployable{Deployment: dep}
	art, err := GuardImage(r, dep)
	if err == nil {
		d.BuildArtifact = art
	}
	return
}

func resolvePair(r Registry, depPair *DeploymentPair) (*DeployablePair, error) {
	prior, _ := resolveName(r, depPair.Prior)
	post, err := resolveName(r, depPair.Post)

	return &DeployablePair{name: depPair.name, Prior: prior, Post: post}, err
}
