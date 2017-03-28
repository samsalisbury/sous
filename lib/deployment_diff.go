package sous

import "strings"

type (
	// DeploymentPair is a pair of deployments that represent a "before and after" style relationship
	DeploymentPair struct {
		name        DeployID
		Prior, Post *Deployment
		Status      DeployStatus
	}
	// DeploymentPairs is a list of DeploymentPair
	DeploymentPairs []*DeploymentPair

	diffSet struct {
		New, Gone, Same, Changed DeploymentPairs
	}

	differ struct {
		from map[DeployID]*DeployState
		DiffChans
	}

	stateDiffer struct {
		from map[DeployID]*DeployState
		DiffChans
	}

	// DiffChans is a set of channels that represent differences between two sets
	// of Deployments as they're discovered
	DiffChans struct {
		Created, Deleted, Retained, Modified chan *DeploymentPair
	}
)

func newDiffSet() diffSet {
	return diffSet{
		New:     DeploymentPairs{},
		Gone:    DeploymentPairs{},
		Same:    DeploymentPairs{},
		Changed: DeploymentPairs{},
	}
}

// ID returns the DeployID of this deployment pair.
func (dp *DeploymentPair) ID() DeployID {
	return dp.name
}

func (d *DiffChans) collect() diffSet {
	ds := newDiffSet()

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
		Created:  make(chan *DeploymentPair, size),
		Deleted:  make(chan *DeploymentPair, size),
		Retained: make(chan *DeploymentPair, size),
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

// Diff computes the differences between two sets of DeployStates
func (d DeployStates) Diff(other Deployments) DiffChans {
	Log.BeChatty()
	difr := newStateDiffer(d)
	go func(d *stateDiffer, o Deployments) {
		e := o.promote(DeployStatusActive)
		d.diff(e)
	}(difr, other)

	return difr.DiffChans
}

// Diff computes the differences between two sets of Deployments
func (d Deployments) Diff(other Deployments) DiffChans {
	difr := newDiffer(d)
	go func(d *differ, o Deployments) {
		d.diff(o)
	}(difr, other)

	return difr.DiffChans
}

func newStateDiffer(intended DeployStates) *stateDiffer {
	i := intended.Snapshot()
	ds := []string{"Computing diff from:"}
	for _, e := range i {
		ds = append(ds, e.String())
	}
	Log.Vomit.Print(strings.Join(ds, "\n    "))

	startMap := make(map[DeployID]*DeployState)
	for _, dep := range i {
		startMap[dep.Name()] = dep
	}
	return &stateDiffer{
		from:      startMap,
		DiffChans: NewDiffChans(len(i)),
	}
}

func newDiffer(intended Deployments) *differ {
	i := intended.Snapshot()
	ds := []string{"Computing diff from:"}
	for _, e := range i {
		ds = append(ds, e.String())
	}
	Log.Vomit.Print(strings.Join(ds, "\n    "))

	startMap := make(map[DeployID]*DeployState)
	for _, dep := range i {
		startMap[dep.Name()] = &DeployState{Deployment: *dep}
	}
	return &differ{
		from:      startMap,
		DiffChans: NewDiffChans(len(i)),
	}
}

func (deps Deployments) promote(all DeployStatus) DeployStates {
	rds := NewDeployStates()
	for _, d := range deps.Snapshot() {
		ds := &DeployState{Deployment: *d, Status: all}
		rds.Add(ds)
	}
	return rds
}

func (d *stateDiffer) diff(existing DeployStates) {
	defer d.DiffChans.Close()
	eds := existing.Snapshot()
	ds := []string{"Computing diff to:"}
	for _, e := range eds {
		ds = append(ds, e.String())
	}
	Log.Vomit.Print(strings.Join(ds, "\n    "))

	for id, existingDS := range existing.Snapshot() {
		intendDS, exists := d.from[id]
		if !exists {

			Log.Debug.Printf("New deployment: %q", id)

			d.Created <- &DeploymentPair{ // XXX s/Created/Create
				name:   id,
				Prior:  nil,
				Post:   &existingDS.Deployment,
				Status: existingDS.Status,
			}
			continue
		}
		delete(d.from, id)
		different, differences := existingDS.Diff(intendDS)

		// This is a bit hacky: if the DSes are different, check to see if they'd
		// be the same if we changed a Pending to Active
		// The purpose here is that for rectification purposes, we should consider
		// "pending" deployments as good as "active" intentions, but we want to
		// report that they're pending in the status
		//
		// The right approach really is to set up a "DeployStatePair", so that both
		// statuses are available and let the rectifier make the determination
		// about what to do.
		if different && intendDS.Status == DeployStatusPending {
			actEx := intendDS.Clone()
			actEx.Status = DeployStatusActive
			different, _ = existingDS.Diff(actEx)
		}

		if different {
			Log.Debug.Printf("Modified deployment: %q (% #v)", id, differences)

			d.Modified <- &DeploymentPair{
				name:  id,
				Prior: &intendDS.Deployment,
				Post:  &existingDS.Deployment,

				// The question of which status to use here implies that this should be
				// a "DeployStatePair", but I'm in a rush
				Status: intendDS.Status,
			}
			continue
		}

		Log.Debug.Printf("Retained deployment: %q (% #v)", id, differences)
		d.Retained <- &DeploymentPair{
			name:   id,
			Prior:  &intendDS.Deployment,
			Post:   &existingDS.Deployment,
			Status: intendDS.Status,
		}
	}

	for _, deletedDS := range d.from {

		Log.Debug.Printf("Deleted deployment: %q", deletedDS.ID())

		d.Deleted <- &DeploymentPair{
			name:   deletedDS.ID(),
			Prior:  &deletedDS.Deployment,
			Post:   nil,
			Status: deletedDS.Status,
		}
	}
}

func (d *differ) diff(existing Deployments) {
	defer d.DiffChans.Close()
	e := existing.Snapshot()
	ds := []string{"Computing diff to:"}
	for _, e := range e {
		ds = append(ds, e.String())
	}
	Log.Vomit.Print(strings.Join(ds, "\n    "))

	for id, existingDeployment := range e {
		intendedDeployment, exists := d.from[id]
		if !exists {

			Log.Debug.Printf("New deployment: %q", id)

			d.Created <- &DeploymentPair{ // XXX s/Created/Create
				name:   id,
				Prior:  nil,
				Post:   existingDeployment,
				Status: DeployStatusAny,
			}
			continue
		}
		delete(d.from, id)
		different, differences := existingDeployment.Diff(&intendedDeployment.Deployment)
		if different {

			Log.Debug.Printf("Modified deployment: %q (% #v)", id, differences)

			d.Modified <- &DeploymentPair{
				name:   id,
				Prior:  &intendedDeployment.Deployment,
				Post:   existingDeployment,
				Status: intendedDeployment.Status,
			}
			continue
		}
		d.Retained <- &DeploymentPair{
			name:   id,
			Prior:  existingDeployment,
			Post:   existingDeployment,
			Status: intendedDeployment.Status,
		}
	}

	for _, deletedDeployment := range d.from {

		Log.Debug.Printf("Deleted deployment: %q", deletedDeployment.ID())

		d.Deleted <- &DeploymentPair{
			name:   deletedDeployment.ID(),
			Prior:  &deletedDeployment.Deployment,
			Post:   nil,
			Status: deletedDeployment.Status,
		}
	}
}
