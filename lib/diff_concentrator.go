package sous

import (
	"sort"
	"sync"

	"github.com/pkg/errors"
)

type (
	// A DiffConcentrator wraps deployment DiffChans in order to produce
	// differences in terms of *manifests*
	DiffConcentrator struct {
		Defs
		Created, Deleted, Retained chan *Manifest
		Modified                   chan *ManifestPair
	}

	ManifestPair struct {
		name        ManifestID
		Prior, Post *Manifest
	}

	deploymentBundle struct {
		consumed bool
		*sync.RWMutex
		pairs map[string]*DeploymentPair
	}
)

// NewDiffChans constructs a DiffChans
func NewConcentrator(defs Defs, s DiffChans, sizes ...int) DiffConcentrator {
	var size int
	if len(sizes) > 0 {
		size = sizes[0]
	}

	return DiffConcentrator{
		Defs:     defs,
		Created:  make(chan *Manifest, size),
		Deleted:  make(chan *Manifest, size),
		Retained: make(chan *Manifest, size),
		Modified: make(chan *ManifestPair, size),
	}
}

func (db *deploymentBundle) add(prior, post *Deployment) error {
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

	if existing, available := db.pairs[cluster]; !available {
		return errors.Errorf(
			"Deployment collision for cluster %q: %v vs %v,%v",
			cluster, existing, prior, post,
		)
	}
	db.Lock()
	defer db.Unlock()

	db.pairs[cluster] = &DeploymentPair{Prior: prior, Post: post}
}

func (db *deploymentBundle) clusters() []string {
	cs := make([]string)
	db.RLock()
	defer db.RUnlock()

	for k := range db.pairs {
		cs = append(cs, k)
	}
	sort.Strings(cs)
	return cs
}

func (db *deploymentBundle) manifestPair(defs Defs) (*ManifestPair, error) {
	before := make(Deployments)
	after := make(Deployments)
	db.RLock()
	defer db.RUnlock()

	for _, v := range db.pairs {
		if v.Prior != nil {
			before = append(before, v.Prior)
		}
		if v.Post != nil {
			after = append(after, v.Post)
		}
	}

	var res *ManifestPair
	ms := before.Manifests(defs)
	switch ms.Len() {
	default:
		return nil, errors.Errorf(
			"bundled deployments produced multiple manifests:\n%#v\n%#v",
			before, ms)
	case 0:
	case 1:
		res.Prior = ms.Get(ms.Keys()[0])
	}

	ms := after.Manifests(defs)
	switch ms.Len() {
	default:
		return nil, errors.Errorf(
			"bundled deployments produced multiple manifests:\n%#v\n%#v",
			after, ms)
	case 0:
	case 1:
		res.Post = ms.Get(ms.Keys()[0])
	}
	db.consumed = true
	return res, nil
}

func newDepBundle() deploymentBundle {
	return deploymentBundle{
		consumed: false,
		RWMutex:  new(sync.RWMutex),
		pairs:    make(map[string]*DeploymentPair),
	}
}

func concentrate(dc DiffChans, con DiffConcentrator) {
	collect := make(map[ManifestID]deploymentBundle)
	addPair = func(mid ManifestID, prior, post *Deployment) {
		depps, present := collect[ManifestID]
		if !present {
			depps := newDepBundle()
		}
		collect[ManifestID].add(prior, post)

		if len(collect[ManifestID].clusters()) == len(con.defs.Clusters) { //eh?
			mp := collect[ManifestID].manifestPair(defs)
			if mp.Prior == nil {
				if mp.Post == nil {
					//?
				} else {
					con.Created <- mp.Post
				}
			} else {
				if mp.Post == nil {
					con.Deleted <- mp.Post
				} else {
					if mp.Prior.Equal(mp.Post) {
						con.Retained <- mp.Post
					} else {
						con.Modified <- mp
					}
				}
			}
		}
	}

	select {
	case c := <-dc.Created:
		addPair(c.ManifestID(), nil, c)
	case d := <-dc.Deleted:
		addPair(c.ManifestID(), d, nil)
	case r := <-dc.Retained:
		addPair(c.ManifestID(), r, r)
	case m := <-dc.Modified:
		addPair(c.ManifestID(), m.Prior, m.Post)
	}

}
