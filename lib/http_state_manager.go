package sous

import (
	"fmt"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
)

type (
	// An HTTPStateManager gets state from a Sous server and transmits updates
	// back to that server.
	HTTPStateManager struct {
		cached   *State
		gdmState restful.Updater
		restful.HTTPClient
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
func (g *gdmWrapper) EmptyReceiver() restful.Comparable {
	return &gdmWrapper{Deployments: []*Deployment{}}
}

// VariancesFrom implements Comparable on gdmWrapper
func (g *gdmWrapper) VariancesFrom(other restful.Comparable) restful.Variances {
	switch og := other.(type) {
	default:
		return restful.Variances{"Not a gdmWrapper"}
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
func NewHTTPStateManager(client restful.HTTPClient) *HTTPStateManager {
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
	messages.ReportLogFieldsMessage("Writing state via HTTP", logging.DebugLevel, logging.Log)
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
	_, err := hsm.Retrieve("./defs", nil, &ds, hsm.User.HTTPHeaders())
	return ds, errors.Wrapf(err, "getting defs")
}

func (hsm *HTTPStateManager) getManifests(defs Defs) (Manifests, error) {
	gdm := gdmWrapper{}
	state, err := hsm.Retrieve("./gdm", nil, &gdm, hsm.User.HTTPHeaders())
	if err != nil {
		return Manifests{}, errors.Wrapf(err, "getting manifests")
	}
	hsm.gdmState = state
	return gdm.manifests(defs)
}

func (hsm *HTTPStateManager) putDeployments(new Deployments) error {
	wNew := wrapDeployments(new)
	_, err := hsm.gdmState.Update(&wNew, hsm.User.HTTPHeaders())
	return errors.Wrapf(err, "putting GDM")
}

// EmptyReceiver implements Comparable on Manifest
func (m *Manifest) EmptyReceiver() restful.Comparable {
	return &Manifest{}
}

// VariancesFrom implements Comparable on Manifest
func (m *Manifest) VariancesFrom(c restful.Comparable) (vs restful.Variances) {
	o, ok := c.(*Manifest)
	if !ok {
		return restful.Variances{fmt.Sprintf("Not a *Manifest: %T", c)}
	}

	_, diffs := m.Diff(o)
	return restful.Variances(diffs)
}
