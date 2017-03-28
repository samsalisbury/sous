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

	DeployablePairs []*DeployablePair

	differ struct {
		from map[DeployID]*DeployState
		*DeployableChans
	}

	stateDiffer struct {
		from map[DeployID]*DeployState
		*DeployableChans
	}
)

func newDiffSet() diffSet {
	return diffSet{
		New:     DeployablePairs{},
		Gone:    DeployablePairs{},
		Same:    DeployablePairs{},
		Changed: DeployablePairs{},
	}
}

// ID returns the DeployID of this deployment pair.
func (dp *DeploymentPair) ID() DeployID {
	return dp.name
}

// Diff computes the differences between two sets of DeployStates
func (d DeployStates) Diff(other Deployments) *DeployableChans {
	difr := newStateDiffer(d)
	go func(d *stateDiffer, o Deployments) {
		e := o.promote(DeployStatusActive)
		d.diff(e)
	}(difr, other)

	return difr.DeployableChans
}

// Diff computes the differences between two sets of Deployments
func (d Deployments) Diff(other Deployments) *DeployableChans {
	difr := newDiffer(d)
	go func(d *differ, o Deployments) {
		d.diff(o)
	}(difr, other)

	return difr.DeployableChans
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
		from:            startMap,
		DeployableChans: NewDeployableChans(len(i)),
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
		from:            startMap,
		DeployableChans: NewDeployableChans(len(i)),
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
	defer d.DeployableChans.Close()
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

			d.Start <- &DeployablePair{ // XXX s/Created/Create
				name:  id,
				Prior: nil,
				Post: &Deployable{
					Deployment: &existingDS.Deployment,
					Status:     existingDS.Status,
				},
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

			d.Update <- &DeployablePair{
				name: id,
				Prior: &Deployable{
					Deployment: &intendDS.Deployment,
					Status:     intendDS.Status,
				},
				Post: &Deployable{
					Deployment: &existingDS.Deployment,
					Status:     existingDS.Status,
				},
			}
			continue
		}

		Log.Debug.Printf("Retained deployment: %q (% #v)", id, differences)
		d.Stable <- &DeployablePair{
			name: id,
			Prior: &Deployable{
				Deployment: &intendDS.Deployment,
				Status:     intendDS.Status,
			},
			Post: &Deployable{
				Deployment: &existingDS.Deployment,
				Status:     existingDS.Status,
			},
		}
	}

	for _, deletedDS := range d.from {

		Log.Debug.Printf("Deleted deployment: %q", deletedDS.ID())

		d.Stop <- &DeployablePair{
			name: deletedDS.ID(),
			Prior: &Deployable{
				Deployment: &deletedDS.Deployment,
				Status:     deletedDS.Status,
			},
			Post: nil,
		}
	}
}

func (d *differ) diff(existing Deployments) {
	defer d.DeployableChans.Close()

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

			d.Start <- &DeployablePair{
				name:  id,
				Prior: nil,
				Post:  &Deployable{Deployment: existingDeployment, Status: DeployStatusActive},
			}
			continue
		}
		delete(d.from, id)
		different, differences := existingDeployment.Diff(&intendedDeployment.Deployment)
		if different {

			Log.Debug.Printf("Modified deployment: %q (% #v)", id, differences)

			d.Update <- &DeployablePair{
				name:  id,
				Prior: &Deployable{Deployment: &intendedDeployment.Deployment, Status: intendedDeployment.Status},
				Post:  &Deployable{Deployment: existingDeployment, Status: DeployStatusActive},
			}
			continue
		}
		d.Stable <- &DeployablePair{
			name:  id,
			Prior: &Deployable{Deployment: &intendedDeployment.Deployment, Status: intendedDeployment.Status},
			Post:  &Deployable{Deployment: &intendedDeployment.Deployment, Status: intendedDeployment.Status},
		}
	}

	for _, deletedDeployment := range d.from {

		Log.Debug.Printf("Deleted deployment: %q", deletedDeployment.ID())

		d.Stop <- &DeployablePair{
			name:  deletedDeployment.ID(),
			Prior: &Deployable{Deployment: &deletedDeployment.Deployment, Status: deletedDeployment.Status},
			Post:  nil,
		}
	}
}
