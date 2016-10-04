package sous

import (
	"log"
	"sort"
	"sync"

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
		*sync.RWMutex
		// XXX this should be two Deployments
		// instead of needing its own mutex
		pairs map[string]*DeploymentPair
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

	log.Printf("conc size: %d", size)

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
		log.Printf("g: %#v", g)
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

	db.Lock()
	log.Printf("%#v", db.pairs)
	if existing, used := db.pairs[cluster]; used {
		return errors.Errorf(
			"Deployment collision for cluster %q: %v vs %v,%v",
			cluster, existing, prior, post,
		)
	}
	defer db.Unlock()

	db.pairs[cluster] = &DeploymentPair{Prior: prior, Post: post}
	return nil
}

func (db *deploymentBundle) clusters() []string {
	log.Printf("pairs: %v", db.pairs)
	cs := make([]string, 0, len(db.pairs))
	db.RLock()
	defer db.RUnlock()

	log.Printf("cls: %v %d", cs, len(cs))
	for k := range db.pairs {
		cs = append(cs, k)
		log.Printf("cls: %v %d", cs, len(cs))
	}
	log.Printf("cls: %v %d", cs, len(cs))
	sort.Strings(cs)
	log.Printf("cls: %v %d", cs, len(cs))
	return cs
}

func (db *deploymentBundle) manifestPair(defs Defs) (*ManifestPair, error) {
	before := NewDeployments()
	after := NewDeployments()
	db.RLock()
	defer db.RUnlock()

	for _, v := range db.pairs {
		if v.Prior != nil {
			before.Add(v.Prior)
		}
		if v.Post != nil {
			after.Add(v.Post)
		}
	}

	res := new(ManifestPair)
	ms, err := before.Manifests(defs)
	log.Printf("%#v", before)
	if err != nil {
		return nil, err
	}
	switch ms.Len() {
	default:
		return nil, errors.Errorf(
			"bundled deployments produced multiple manifests:\n%#v\n%#v",
			before, ms)
	case 0:
	case 1:
		p, got := ms.Get(ms.Keys()[0])
		if !got {
			panic("Non-empty Manifests returned no value for a reported key")
		}
		res.Prior = p
	}

	ms, err = after.Manifests(defs)
	log.Printf("%#v", after)
	if err != nil {
		return nil, err
	}
	switch ms.Len() {
	default:
		return nil, errors.Errorf(
			"bundled deployments produced multiple manifests:\n%#v\n%#v",
			after, ms)
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

	db.consumed = true
	return res, nil
}

func newDepBundle() *deploymentBundle {
	return &deploymentBundle{
		consumed: false,
		RWMutex:  new(sync.RWMutex),
		pairs:    make(map[string]*DeploymentPair),
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
		log.Printf("addPair: %#v: %#v %#v", mid, prior, post)
		_, present := collect[mid]
		if !present {
			collect[mid] = newDepBundle()
		}

		err := collect[mid].add(prior, post)
		if err != nil {
			log.Printf("err: %#v", err)
			con.Errors <- err
			return
		}

		log.Printf("Can dispatch? %v ?= %v", len(collect[mid].clusters()), len(con.Defs.Clusters))
		if len(collect[mid].clusters()) == len(con.Defs.Clusters) { //eh?
			mp, err := collect[mid].manifestPair(con.Defs)
			if err != nil {
				con.Errors <- err
				return
			}
			log.Printf("Dispatching %#v", mp)
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
			addPair(m.Prior.ManifestID(), m.Prior, m.Post)
		}
	}
}
