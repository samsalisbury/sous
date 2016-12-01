package sous

import "fmt"

type (
	// A DeployableChans is a bundle of channels describing actions to take on a
	// cluster
	DeployableChans struct {
		Start, Stop, Stable chan *Deployable
		Update              chan *DeployablePair
	}

	// A DeployablePair is a pair of deployables, describing a "before and after"
	// situation, where the Prior Deployable is the known state and the Post
	// Deployable is the desired state.
	DeployablePair struct {
		Prior, Post *Deployable
		name        DeployID
	}
)

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

func (dp *DeployablePair) ID() DeployID {
	return dp.name
}

func (dc *DeployableChans) ResolveNames(r Registry, diff *DiffChans, errs chan error) {
	go resolveSingles(r, diff.Created, dc.Start, errs)
	go unresolvedSingles(r, diff.Deleted, dc.Stop, errs)
	go unresolvedSingles(r, diff.Retained, dc.Stable, errs)
	go resolvePairs(r, diff.Modified, dc.Update, errs)
}

func unresolvedSingles(r Registry, from chan *Deployment, to chan *Deployable, errs chan error) {
	for dep := range from {
		unresolvedSingle(r, dep, to, errs)
	}
	close(to)
}

func unresolvedSingle(r Registry, dep *Deployment, to chan *Deployable, errs chan error) {
	art, err := GuardImage(r, dep)
	if err != nil {
		art = nil // we can live without build artifacts for stops and stables
	}
	d := &Deployable{
		Deployment:    dep,
		BuildArtifact: art,
	}
	to <- d
}

func resolveSingles(r Registry, from chan *Deployment, to chan *Deployable, errs chan error) {
	for dep := range from {
		resolveSingle(r, dep, to, errs)
	}
	close(to)
}

func resolveSingle(r Registry, dep *Deployment, to chan *Deployable, errs chan error) {
	art, err := GuardImage(r, dep)
	if err != nil {
		errs <- err
		return
	}
	if art == nil {
		return // nil error and artifact == not deploying
	}
	d := &Deployable{
		Deployment:    dep,
		BuildArtifact: art,
	}
	to <- d
}

func resolvePairs(r Registry, from chan *DeploymentPair, to chan *DeployablePair, errs chan error) {
	for depPair := range from {
		resolvePair(r, depPair, to, errs)
	}
	close(to)
}

func resolvePair(r Registry, depPair *DeploymentPair, to chan *DeployablePair, errs chan error) {
	priorArt, err := GuardImage(r, depPair.Prior)
	if err != nil {
		priorArt = nil // usually not a blocker
	}

	art, err := GuardImage(r, depPair.Post)
	if err != nil {
		errs <- err
		return
	}
	if art == nil {
		return // nil error and artifact == not deploying
	}

	d := &DeployablePair{
		name: depPair.name,
		Prior: &Deployable{
			Deployment:    depPair.Prior,
			BuildArtifact: priorArt,
		},
		Post: &Deployable{
			Deployment:    depPair.Post,
			BuildArtifact: art,
		},
	}
	to <- d
}
