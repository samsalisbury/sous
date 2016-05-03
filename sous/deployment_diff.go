package sous

type (
	DeploymentPair struct {
		name        DepName
		prior, post *Deployment
	}
	DeploymentPairs []DeploymentPair

	DiffSet struct {
		New, Gone, Same Deployments
		Changed         DeploymentPairs
	}

	differ struct {
		from map[DepName]Deployment
		DiffChans
	}

	DiffChans struct {
		Created, Deleted, Retained chan Deployment
		Modified                   chan DeploymentPair
	}
)

func (dc *DiffChans) Collect() DiffSet {
	ds := DiffSet{
		make(Deployments, 0),
		make(Deployments, 0),
		make(Deployments, 0),
		make(DeploymentPairs, 0),
	}

	for g := range dc.Deleted {
		ds.Gone = append(ds.Gone, g)
	}
	for n := range dc.Created {
		ds.New = append(ds.New, n)
	}
	for m := range dc.Modified {
		ds.Changed = append(ds.Changed, m)
	}
	for s := range dc.Retained {
		ds.Same = append(ds.Same, s)
	}
	return ds
}

func NewDiffChans(sizes ...int) DiffChans {
	var size int
	if len(sizes) > 0 {
		size = sizes[0]
	}

	return DiffChans{
		Created:  make(chan Deployment, size),
		Deleted:  make(chan Deployment, size),
		Retained: make(chan Deployment, size),
		Modified: make(chan DeploymentPair, size),
	}
}

func (d *DiffChans) Close() {
	close(d.Created)
	close(d.Retained)
	close(d.Modified)
	close(d.Deleted)
}

func (d Deployments) Diff(other Deployments) DiffChans {
	difr := newDiffer(d)
	go func(d *differ, o Deployments) {
		d.diff(o)
	}(difr, other)

	return difr.DiffChans
}

func newDiffer(intended Deployments) *differ {
	startMap := make(map[DepName]Deployment)
	for _, dep := range intended {
		startMap[dep.Name()] = dep
	}
	return &differ{
		from:      startMap,
		DiffChans: NewDiffChans(len(intended)),
	}
}

func (d *differ) diff(existing Deployments) {
	for i := range existing {
		name := existing[i].Name()
		if indep, ok := d.from[name]; ok {
			delete(d.from, name)
			if indep.Equal(existing[i]) {
				d.Retained <- indep
			} else {
				d.Modified <- DeploymentPair{name, &existing[i], &indep}
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
