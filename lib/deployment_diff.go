package sous

import (
	"fmt"
	"strings"
)

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

// Diff computes the differences between two sets of Deployments
func (d DeployStates) Diff(other Deployments) DiffChans {
	differ := newStateDiffer(d)
	go differ.diff(other)
	return differ.DiffChans
}

// Diff computes the differences between two sets of Deployments
func (d Deployments) Diff(other Deployments) DiffChans {
	differ := newDiffer(d)
	go differ.diff(other)
	return differ.DiffChans
}

func newStateDiffer(intended DeployStates) *differ {
	intended = intended.Clone()
	logDeployStates(intended, "intended deploy states")
	return &differ{
		from:      intended.Snapshot(),
		DiffChans: NewDiffChans(intended.Len()),
	}
}

func newDiffer(intendedDeployments Deployments) *differ {
	intended := intendedDeployments.Clone().ToDeployStatesWithStatus(DeployStatusSucceeded)
	logDeployStates(intended, "intended deployments")
	return &differ{
		from:      intended.Snapshot(),
		DiffChans: NewDiffChans(intended.Len()),
	}
}

func logDeployStates(dss DeployStates, desc string) {
	message := []string{fmt.Sprintf("Computing diff from %s:", desc)}
	for _, ds := range dss.Snapshot() {
		message = append(message, ds.String())
	}
	Log.Vomit.Print(strings.Join(message, "\n    "))
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
