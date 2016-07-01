package sous

import "strings"

type (
	// DeploymentPair is a pair of deployments that represent a "before and after" style relationship
	DeploymentPair struct {
		name        DepName
		prior, post *Deployment
	}
	// DeploymentPairs is a list of DeploymentPair
	DeploymentPairs []*DeploymentPair

	diffSet struct {
		New, Gone, Same Deployments
		Changed         DeploymentPairs
	}

	differ struct {
		from map[DepName]*Deployment
		DiffChans
	}

	// DiffChans is a set of channels that represent differences between two sets
	// of Deployments as they're discovered
	DiffChans struct {
		Created, Deleted, Retained chan *Deployment
		Modified                   chan *DeploymentPair
	}
)

func (d *DiffChans) collect() diffSet {
	ds := diffSet{
		make(Deployments, 0),
		make(Deployments, 0),
		make(Deployments, 0),
		make(DeploymentPairs, 0),
	}

	for g := range d.Deleted {
		ds.Gone = append(ds.Gone, g)
	}
	for n := range d.Created {
		ds.New = append(ds.New, n)
	}
	for m := range d.Modified {
		ds.Changed = append(ds.Changed, m)
	}
	for s := range d.Retained {
		ds.Same = append(ds.Same, s)
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
	ds := []string{"Computing diff from:"}
	for _, e := range intended {
		ds = append(ds, e.String())
	}
	Log.Debug.Print(strings.Join(ds, "\n    "))

	startMap := make(map[DepName]*Deployment)
	for _, dep := range intended {
		startMap[dep.Name()] = dep
	}
	return &differ{
		from:      startMap,
		DiffChans: NewDiffChans(len(intended)),
	}
}

func (d *differ) diff(existing Deployments) {
	ds := []string{"Computing diff to:"}
	for _, e := range existing {
		ds = append(ds, e.String())
	}
	Log.Debug.Print(strings.Join(ds, "\n    "))

	for i := range existing {
		name := existing[i].Name()
		if indep, ok := d.from[name]; ok {
			delete(d.from, name)
			if indep.Equal(existing[i]) {
				d.Retained <- indep
			} else {
				d.Modified <- &DeploymentPair{name, indep, existing[i]}
			}
		} else {
			d.Created <- existing[i]
		}
	}

	for _, dep := range d.from {
		d.Deleted <- dep
	}

	d.DiffChans.Close()
}
