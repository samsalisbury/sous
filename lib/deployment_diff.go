package sous

import "strings"

type (
	// DeploymentPair is a pair of deployments that represent a "before and after" style relationship
	DeploymentPair struct {
		name        DeployID
		Prior, Post *Deployment
	}
	// DeploymentPairs is a list of DeploymentPair
	DeploymentPairs []*DeploymentPair

	diffSet struct {
		New, Gone, Same Deployments
		Changed         DeploymentPairs
	}

	differ struct {
		from map[DeployID]*Deployment
		DiffChans
	}

	// DiffChans is a set of channels that represent differences between two sets
	// of Deployments as they're discovered
	DiffChans struct {
		Created, Deleted, Retained chan *Deployment
		Modified                   chan *DeploymentPair
	}
)

func newDiffSet() diffSet {
	return diffSet{
		New:     NewDeployments(),
		Gone:    NewDeployments(),
		Same:    NewDeployments(),
		Changed: make(DeploymentPairs, 0),
	}
}

// ID returns the DeployID of this deployment pair.
func (dp *DeploymentPair) ID() DeployID {
	return dp.name
}

func (d *DiffChans) collect() diffSet {
	ds := newDiffSet()

	for g := range d.Deleted {
		ds.Gone.Add(g)
	}
	for n := range d.Created {
		ds.New.Add(n)
	}
	for m := range d.Modified {
		ds.Changed = append(ds.Changed, m)
	}
	for s := range d.Retained {
		ds.Same.Add(s)
	}
	return ds
}

// NewDiffChans constructs a DiffChans
func NewDiffChans(sizes ...int) DiffChans {
	var size int
	if len(sizes) > 0 {
		size = sizes[0]
	}

	return DiffChans{
		Created:  make(chan *Deployment, size),
		Deleted:  make(chan *Deployment, size),
		Retained: make(chan *Deployment, size),
		Modified: make(chan *DeploymentPair, size),
	}
}

// Close closes all the channels in a DiffChans in a single action
func (d *DiffChans) Close() {
	close(d.Created)
	close(d.Retained)
	close(d.Modified)
	close(d.Deleted)
}

// Diff computes the differences between two sets of Deployments
func (d Deployments) Diff(other Deployments) DiffChans {
	difr := newDiffer(d)
	go func(d *differ, o Deployments) {
		d.diff(o)
	}(difr, other)

	return difr.DiffChans
}

func newDiffer(intended Deployments) *differ {
	i := intended.Snapshot()
	ds := []string{"Computing diff from:"}
	for _, e := range i {
		ds = append(ds, e.String())
	}
	Log.Debug.Print(strings.Join(ds, "\n    "))

	startMap := make(map[DeployID]*Deployment)
	for _, dep := range i {
		startMap[dep.Name()] = dep
	}
	return &differ{
		from:      startMap,
		DiffChans: NewDiffChans(len(i)),
	}
}

func (d *differ) diff(existing Deployments) {
	defer d.DiffChans.Close()
	e := existing.Snapshot()
	ds := []string{"Computing diff to:"}
	for _, e := range e {
		ds = append(ds, e.String())
	}
	Log.Debug.Print(strings.Join(ds, "\n    "))

	for id, existingDeployment := range e {
		intendedDeployment, exists := d.from[id]
		if !exists {

			Log.Debug.Printf("New deployment: %q", id)

			d.Created <- existingDeployment
			continue
		}
		delete(d.from, id)
		different, differences := existingDeployment.Diff(intendedDeployment)
		if different {

			Log.Debug.Printf("Modified deployment: %q (% #v)", id, differences)

			d.Modified <- &DeploymentPair{
				name:  id,
				Prior: intendedDeployment,
				Post:  existingDeployment,
			}
			continue
		}
		d.Retained <- existingDeployment
	}

	for _, deletedDeployment := range d.from {

		Log.Debug.Printf("Deleted deployment: %q", deletedDeployment.ID())

		d.Deleted <- deletedDeployment
	}
}
