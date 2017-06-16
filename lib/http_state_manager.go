package sous

import (
	"fmt"

	"github.com/pkg/errors"
)

type (
	// An HTTPStateManager gets state from a Sous server and transmits updates
	// back to that server.
	HTTPStateManager struct {
		cached   *State
		gdmState *resourceState
		HTTPClient
		User User
	}

	gdmWrapper struct {
		Deployments []*Deployment
	}
)

func wrapDeployments(source Deployments) gdmWrapper {
	data := gdmWrapper{Deployments: make([]*Deployment, 0)}
	for _, d := range source.Snapshot() {
		data.Deployments = append(data.Deployments, d)
	}
	return data
}

// EmptyReceiver implements Comparable on gdmWrapper
func (g *gdmWrapper) EmptyReceiver() Comparable {
	return &gdmWrapper{Deployments: []*Deployment{}}
}

// VariancesFrom implements Comparable on gdmWrapper
func (g *gdmWrapper) VariancesFrom(other Comparable) Variances {
	switch og := other.(type) {
	default:
		return Variances{"Not a gdmWrapper"}
	case *gdmWrapper:
		return g.unwrap().VariancesFrom(og.unwrap())
	}
}

func (g *gdmWrapper) unwrap() *Deployments {
	ds := NewDeployments(g.Deployments...)
	return &ds
}

func (g *gdmWrapper) manifests(defs Defs) (Manifests, error) {
	ds := NewDeployments()
	for _, d := range g.Deployments {
		ds.Add(d)
	}
	return ds.RawManifests(defs)
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
	if hsm.gdmState == nil {
		_, err := hsm.ReadState()
		if err != nil {
			return err
		}
	}

	wds, err := s.Deployments()
	if err != nil {
		return err
	}

	return hsm.putDeployments(wds)
}

////

func (hsm *HTTPStateManager) getDefs() (Defs, error) {
	ds := Defs{}
	return ds, errors.Wrapf(hsm.Retrieve("./defs", nil, &ds, hsm.User), "getting defs")
}

func (hsm *HTTPStateManager) getManifests(defs Defs) (Manifests, error) {
	gdm := gdmWrapper{}
	state, err := hsm.RetrieveWithState("./gdm", nil, &gdm, hsm.User)
	if err != nil {
		return Manifests{}, errors.Wrapf(err, "getting manifests")
	}
	hsm.gdmState = state
	return gdm.manifests(defs)
}

func (hsm *HTTPStateManager) putDeployments(new Deployments) error {
	wNew := wrapDeployments(new)
	return errors.Wrapf(hsm.Update("./gdm", nil, hsm.gdmState, &wNew, hsm.User), "putting GDM")
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
