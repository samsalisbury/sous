package sous

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

type nameResolver struct {
	registry Registry
}

// ResolveNames resolves diffs.
func (d *DeployableChans) ResolveNames(ctx context.Context, r Registry) *DeployableChans {
	names := &nameResolver{registry: r}

	return d.Pipeline(ctx, names)
}

func (names *nameResolver) Start(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
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
	art, err := guardImage(r, d.Deployment)
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

func guardImage(r Registry, d *Deployment) (*BuildArtifact, error) {
	if d.NumInstances == 0 {
		logging.Log.Info.Printf("Deployment %q has 0 instances, skipping artifact check.", d.ID())
		return nil, nil
	}
	art, err := r.GetArtifact(d.SourceID)
	spew.Dump("resolving", d.SourceID, art, err)
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
