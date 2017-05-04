package sous

import (
	"fmt"

	"github.com/pkg/errors"
)

type (
	// An HTTPStateManager gets state from a Sous server and transmits updates
	// back to that server.
	HTTPStateManager struct {
		cached *State
		HTTPClient
		User User
	}

	gdmWrapper struct {
		Deployments []*Deployment
	}
)

func (g *gdmWrapper) manifests(defs Defs, base Manifests) (Manifests, error) {
	ds := NewDeployments()
	for _, d := range g.Deployments {
		ds.Add(d)
	}
	return ds.PutbackManifests(defs, base)
}

// NewHTTPStateManager creates a new HTTPStateManager.
func NewHTTPStateManager(client HTTPClient) *HTTPStateManager {
	return &HTTPStateManager{HTTPClient: client}
}

// ReadState implements StateReader for HTTPStateManager.
func (hsm *HTTPStateManager) ReadState() (*State, error) {
	defs, err := hsm.getDefs()
	if err != nil {
		return nil, err
	}
	ms, err := hsm.getManifests(defs)
	if err != nil {
		return nil, err
	}

	hsm.cached = &State{
		Defs:      defs,
		Manifests: ms,
	}
	return hsm.cached.Clone(), nil
}

// WriteState implements StateWriter for HTTPStateManager.
func (hsm *HTTPStateManager) WriteState(s *State, u User) error {
	hsm.User = u
	flaws := s.Validate()
	if len(flaws) > 0 {
		return errors.Errorf("Invalid update to state: %v", flaws)
	}
	Log.Debug.Printf("Writing state via HTTP.")
	if hsm.cached == nil {
		_, err := hsm.ReadState()
		if err != nil {
			return err
		}
	}
	wds, err := s.Deployments()
	if err != nil {
		return err
	}
	cds, err := hsm.cached.Deployments()
	if err != nil {
		return err
	}
	diff := cds.Diff(wds)
	cchs := diff.Concentrate(s.Defs, hsm.cached.Manifests)
	Log.Debug.Printf("Processing diffs...")
	return hsm.process(cchs)
}

func (hsm *HTTPStateManager) process(dc DiffConcentrator) error {
	done := make(chan struct{})
	defer close(done)

	createErrs := make(chan error)
	go hsm.creates(dc.Created, createErrs, done)

	deleteErrs := make(chan error)
	go hsm.deletes(dc.Deleted, deleteErrs, done)

	modifyErrs := make(chan error)
	go hsm.modifies(dc.Modified, modifyErrs, done)

	retainErrs := make(chan error)
	go hsm.retains(dc.Retained, retainErrs, done)

	for {
		if createErrs == nil && deleteErrs == nil && modifyErrs == nil && retainErrs == nil {
			return nil
		}

		select {
		case e, open := <-dc.Errors:
			if open {
				return e
			}
			dc.Errors = nil
		case e, open := <-createErrs:
			if open {
				return e
			}
			createErrs = nil
		case e, open := <-deleteErrs:
			if open {
				return e
			}
			deleteErrs = nil
		case e, open := <-retainErrs:
			if open {
				return e
			}
			retainErrs = nil
		case e, open := <-modifyErrs:
			if open {
				return e
			}
			modifyErrs = nil
		}
	}
}

func (hsm *HTTPStateManager) retains(mc chan *Manifest, ec chan error, done chan struct{}) {
	defer close(ec)
	for {
		select {
		case <-done:
			return
		case _, open := <-mc: //just drop 'em
			if !open {
				return
			}
		}
	}
}

func (hsm *HTTPStateManager) creates(mc chan *Manifest, ec chan error, done chan struct{}) {
	defer close(ec)
	for {
		select {
		case <-done:
			return
		case m, open := <-mc:
			if !open {
				return
			}
			if err := hsm.create(m); err != nil {
				ec <- err
			}
		}
	}
}

func (hsm *HTTPStateManager) deletes(mc chan *Manifest, ec chan error, done chan struct{}) {
	defer close(ec)
	for {
		select {
		case <-done:
			return
		case m, open := <-mc:
			if !open {
				return
			}
			if err := hsm.del(m); err != nil {
				ec <- err
			}
		}
	}
}

func (hsm *HTTPStateManager) modifies(mc chan *ManifestPair, ec chan error, done chan struct{}) {
	defer close(ec)
	for {
		select {
		case <-done:
			return
		case m, open := <-mc:
			if !open {
				return
			}
			Log.Debug.Printf("Modifying %q", m.name)
			if err := hsm.modify(m); err != nil {
				ec <- err
			}
		}
	}
}

////

func (hsm *HTTPStateManager) getDefs() (Defs, error) {
	ds := Defs{}
	return ds, errors.Wrapf(hsm.Retrieve("./defs", nil, &ds, hsm.User), "getting defs")
}

func (hsm *HTTPStateManager) getManifests(defs Defs) (Manifests, error) {
	gdm := gdmWrapper{}
	if err := hsm.Retrieve("./gdm", nil, &gdm, hsm.User); err != nil {
		return Manifests{}, errors.Wrapf(err, "getting manifests")
	}
	return gdm.manifests(defs)
}

func manifestParams(m *Manifest) map[string]string {
	return map[string]string{
		"repo":   m.Source.Repo,
		"offset": m.Source.Dir,
		"flavor": m.Flavor,
	}
}

func manifestDebugs(m *Manifest) (r, o, f string) {
	return m.Source.Repo, m.Source.Dir, m.Flavor
}

// EmptyReceiver implements Comparable on Manifest
func (m *Manifest) EmptyReceiver() Comparable {
	return &Manifest{}
}

// VariancesFrom implements Comparable on Manifest
func (m *Manifest) VariancesFrom(c Comparable) (vs Variances) {
	o, ok := c.(*Manifest)
	if !ok {
		return Variances{fmt.Sprintf("Not a *Manifest: %T", c)}
	}

	_, diffs := m.Diff(o)
	return Variances(diffs)
}

func (hsm *HTTPStateManager) create(m *Manifest) error {
	r, o, f := manifestDebugs(m)
	return errors.Wrapf(hsm.Create("./manifest", manifestParams(m), m, hsm.User), "creating manifest %s %s %s", r, o, f)
}

func (hsm *HTTPStateManager) del(m *Manifest) error {
	r, o, f := manifestDebugs(m)
	return errors.Wrapf(hsm.Delete("./manifest", manifestParams(m), m, hsm.User), "deleting manifest %s %s %s", r, o, f)
}

func (hsm *HTTPStateManager) modify(mp *ManifestPair) error {
	bf, af := mp.Prior, mp.Post
	r, o, f := manifestDebugs(bf)
	return errors.Wrapf(hsm.Update("./manifest", manifestParams(bf), bf, af, hsm.User), "updating manifest %s %s %s", r, o, f)
}
