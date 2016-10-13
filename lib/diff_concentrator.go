package sous

import (
	"sort"

	"github.com/pkg/errors"
)

type (
	ManifestPair struct {
		name        ManifestID
		Prior, Post *Manifest
	}
	ManifestPairs []*ManifestPair

	// A DiffConcentrator wraps deployment DiffChans in order to produce
	// differences in terms of *manifests*
	DiffConcentrator struct {
		Defs
		Errors                     chan error
		Created, Deleted, Retained chan *Manifest
		Modified                   chan *ManifestPair
	}

	concentratedDiffSet struct {
		New, Gone, Same Manifests
		Changed         ManifestPairs
	}

	deploymentBundle struct {
		consumed bool
		before   Deployments
		after    Deployments
	}
)

// Concentrate returns a DiffConcentrator set up to concentrate the deployment
// changes in a DiffChans into manifest changes
func (d DiffChans) Concentrate(defs Defs) DiffConcentrator {
	c := NewConcentrator(defs, d, cap(d.Created))
	go concentrate(d, c)
	return c
}

// NewDiffChans constructs a DiffChans
func NewConcentrator(defs Defs, s DiffChans, sizes ...int) DiffConcentrator {
	var size int
	if len(sizes) > 0 {
		size = sizes[0]
	}

	return DiffConcentrator{
		Defs:     defs,
		Errors:   make(chan error, size+10),
		Created:  make(chan *Manifest, size),
		Deleted:  make(chan *Manifest, size),
		Retained: make(chan *Manifest, size),
		Modified: make(chan *ManifestPair, size),
	}
}

func newConcDiffSet() concentratedDiffSet {
	return concentratedDiffSet{
		New:     NewManifests(),
		Gone:    NewManifests(),
		Same:    NewManifests(),
		Changed: make(ManifestPairs, 0),
	}
}

func (dc *DiffConcentrator) collect() (concentratedDiffSet, error) {
	ds := newConcDiffSet()

	select {
	default:
	case err := <-dc.Errors:
		return ds, err
	}
	for g := range dc.Deleted {
		ds.Gone.Add(g)
	}
	for n := range dc.Created {
		ds.New.Add(n)
	}
	for m := range dc.Modified {
		ds.Changed = append(ds.Changed, m)
	}
	for s := range dc.Retained {
		ds.Same.Add(s)
	}
	select {
	default:
	case err := <-dc.Errors:
		return ds, err
	}

	return ds, nil
}

func (db *deploymentBundle) add(prior, post *Deployment) error {
	if db.consumed {
		return errors.Errorf("Attempted to add a new pair to a consumed bundle: %v %v", prior, post)
	}
	var cluster string
	if prior != nil {
		cluster = prior.ClusterName
	}
	if post != nil {
		if prior == nil {
			cluster = post.ClusterName
		} else if cluster != post.ClusterName {
			return errors.Errorf("Invariant violated: two clusters named in deploy pair: %q vs %q", prior.ClusterName, post.ClusterName)
		}
	}
	if cluster == "" {
		return errors.Errorf("Invariant violated: no cluster name given in deploy pair")
	}

	if prior != nil {
		if accepted := db.before.Add(prior); !accepted {
			existing, present := db.before.Get(prior.ID())
			if !present {
				panic("Collided deployment not present!")
			}
			return errors.Errorf(
				"Deployment collision for cluster's prior %q:\n  %v vs\n  %v",
				cluster, existing, prior,
			)
		}
	}

	if post != nil {
		if accepted := db.after.Add(post); !accepted {
			existing, present := db.after.Get(post.ID())
			if !present {
				panic("Collided deployment not present!")
			}
			return errors.Errorf(
				"Deployment collision for cluster's post %q:\n  %v vs\n  %v",
				cluster, existing, post,
			)
		}
	}

	return nil
}

func (db *deploymentBundle) clusters() []string {
	cm := make(map[string]struct{})
	for _, v := range db.before.Snapshot() {
		cm[v.ClusterName] = struct{}{}
	}
	for _, v := range db.after.Snapshot() {
		cm[v.ClusterName] = struct{}{}
	}
	cs := make([]string, 0, len(cm))
	for k := range cm {
		cs = append(cs, k)
	}
	sort.Strings(cs)
	return cs
}

func (db *deploymentBundle) manifestPair(defs Defs) (*ManifestPair, error) {
	db.consumed = true
	res := new(ManifestPair)
	ms, err := db.before.Manifests(defs)
	if err != nil {
		return nil, err
	}
	switch ms.Len() {
	default:
		return nil, errors.Errorf(
			"bundled deployments produced multiple manifests:\n%#v\n%#v",
			db.before, ms)
	case 0:
	case 1:
		p, got := ms.Get(ms.Keys()[0])
		if !got {
			panic("Non-empty Manifests returned no value for a reported key")
		}
		res.Prior = p
	}

	ms, err = db.after.Manifests(defs)
	if err != nil {
		return nil, err
	}
	switch ms.Len() {
	default:
		return nil, errors.Errorf(
			"bundled deployments produced multiple manifests:\n%#v\n%#v",
			db.after, ms)
	case 0:
	case 1:
		p, got := ms.Get(ms.Keys()[0])
		if !got {
			panic("Non-empty Manifests returned no value for a reported key")
		}
		res.Post = p
	}
	if res.Post == nil {
		res.name = res.Prior.ID()
	} else {
		res.name = res.Post.ID()
	}

	return res, nil
}

func newDepBundle() *deploymentBundle {
	return &deploymentBundle{
		consumed: false,
		before:   NewDeployments(),
		after:    NewDeployments(),
	}
}

func (dc *DiffConcentrator) dispatch(mp *ManifestPair) error {
	if mp.Prior == nil {
		if mp.Post == nil {
			return errors.Errorf("Blank manifest pair: %#v", mp)
		}
		dc.Created <- mp.Post
	} else {
		if mp.Post == nil {
			dc.Deleted <- mp.Prior
		} else {
			if mp.Prior.Equal(mp.Post) {
				dc.Retained <- mp.Post
			} else {
				dc.Modified <- mp
			}
		}
	}
	return nil
}

func concentrate(dc DiffChans, con DiffConcentrator) {
	collect := make(map[ManifestID]*deploymentBundle)
	addPair := func(mid ManifestID, prior, post *Deployment) {
		_, present := collect[mid]
		if !present {
			collect[mid] = newDepBundle()
		}

		err := collect[mid].add(prior, post)
		if err != nil {
			con.Errors <- err
			return
		}

		if len(collect[mid].clusters()) == len(con.Defs.Clusters) { //eh?
			mp, err := collect[mid].manifestPair(con.Defs)
			if err != nil {
				con.Errors <- err
				return
			}
			if err := con.dispatch(mp); err != nil {
				con.Errors <- err
			}
		}
	}

	created, deleted, retained, modified :=
		dc.Created, dc.Deleted, dc.Retained, dc.Modified

	for {
		if created == nil && deleted == nil && retained == nil && modified == nil {
			close(con.Retained)
			close(con.Modified)
			close(con.Errors)
			break
		}

		select {
		case c, open := <-created:
			if !open {
				close(con.Created)
				created = nil
				continue
			}
			addPair(c.ManifestID(), nil, c)
		case d, open := <-deleted:
			if !open {
				close(con.Deleted)
				deleted = nil
				continue
			}
			addPair(d.ManifestID(), d, nil)
		case r, open := <-retained:
			if !open {
				retained = nil
				continue
			}
			addPair(r.ManifestID(), r, r)
		case m, open := <-modified:
			if !open {
				modified = nil
				continue
			}
			addPair(m.Prior.ManifestID(), m.Post, m.Prior)
		}
	}
}
