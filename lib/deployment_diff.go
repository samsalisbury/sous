package sous

import (
	"strings"

	"github.com/opentable/sous/util/logging"
)

type (
	// DeploymentPair is a pair of deployments that represent a "before and after" style relationship
	DeploymentPair struct {
		name        DeploymentID
		Prior, Post *Deployment
		Diffs       Differences
		Status      DeployStatus
	}
	// DeploymentPairs is a list of DeploymentPair
	DeploymentPairs []*DeploymentPair

	// DeployablePairs is a list of DeployablePair
	DeployablePairs []*DeployablePair

	differ struct {
		from map[DeploymentID]*DeployState
		*DeployableChans
	}

	stateDiffer struct {
		from map[DeploymentID]*DeployState
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
func (dp *DeploymentPair) ID() DeploymentID {
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
	logging.Log.Vomit.Print(strings.Join(ds, "\n    "))

	startMap := make(map[DeploymentID]*DeployState)
	for _, dep := range i {
		startMap[dep.Name()] = dep
	}
	chans := NewDeployableChans(len(i))

	return &stateDiffer{
		from:            startMap,
		DeployableChans: chans,
	}
}

func newDiffer(intended Deployments) *differ {
	i := intended.Snapshot()
	ds := []string{"Computing diff from:"}
	for _, e := range i {
		ds = append(ds, e.String())
	}
	logging.Log.Vomit.Print(strings.Join(ds, "\n    "))

	startMap := make(map[DeploymentID]*DeployState)
	for _, dep := range i {
		startMap[dep.Name()] = &DeployState{Deployment: *dep}
	}
	chans := NewDeployableChans(len(i))

	return &differ{
		from:            startMap,
		DeployableChans: chans,
	}
}

func (d Deployments) promote(all DeployStatus) DeployStates {
	rds := NewDeployStates()
	for _, ad := range d.Snapshot() {
		ds := &DeployState{Deployment: *ad, Status: all}
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
	logging.Log.Vomit.Print(strings.Join(ds, "\n    "))

	for id, existingDS := range existing.Snapshot() {
		intendDS, exists := d.from[id]
		if !exists {

			logging.Log.Debug.Printf("New deployment: %q", id)

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

		if different {
			logging.Log.Debug.Printf("Modified deployment: %q (% #v)", id, differences)

			d.Update <- &DeployablePair{
				name:         id,
				Diffs:        differences,
				ExecutorData: intendDS.ExecutorData,
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

		logging.Log.Debug.Printf("Retained deployment: %q (% #v)", id, differences)
		d.Stable <- &DeployablePair{
			name:         id,
			Diffs:        Differences{},
			ExecutorData: intendDS.ExecutorData,
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

		logging.Log.Debug.Printf("Deleted deployment: %q", deletedDS.ID())

		d.Stop <- &DeployablePair{
			name:         deletedDS.ID(),
			ExecutorData: deletedDS.ExecutorData,
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
	logging.Log.Vomit.Print(strings.Join(ds, "\n    "))

	for id, existingDeployment := range e {
		intendedDeployment, exists := d.from[id]
		if !exists {

			logging.Log.Debug.Printf("New deployment: %q", id)

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

			logging.Log.Debug.Printf("Modified deployment: %q (% #v)", id, differences)

			d.Update <- &DeployablePair{
				name:  id,
				Diffs: differences,
				Prior: &Deployable{Deployment: &intendedDeployment.Deployment, Status: intendedDeployment.Status},
				Post:  &Deployable{Deployment: existingDeployment, Status: DeployStatusActive},
			}
			continue
		}
		d.Stable <- &DeployablePair{
			name:  id,
			Diffs: Differences{},
			Prior: &Deployable{Deployment: &intendedDeployment.Deployment, Status: intendedDeployment.Status},
			Post:  &Deployable{Deployment: &intendedDeployment.Deployment, Status: intendedDeployment.Status},
		}
	}

	for _, deletedDeployment := range d.from {

		logging.Log.Debug.Printf("Deleted deployment: %q", deletedDeployment.ID())

		d.Stop <- &DeployablePair{
			name:  deletedDeployment.ID(),
			Prior: &Deployable{Deployment: &deletedDeployment.Deployment, Status: deletedDeployment.Status},
			Post:  nil,
		}
	}
}
