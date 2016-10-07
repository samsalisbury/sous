package sous

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type (
	HTTPStateManager struct {
		serverURL url.URL
		cached    *State
		http.Client
	}

	gdmWrapper struct {
		Deployments []*Deployment
	}
)

func (g *gdmWrapper) manifests() {
	ds := NewDeployments()
	for d := range g.Deployments {
		ds.Add(d)
	}
	return d.Manifests()
}

func (g *gdmWrapper) fromJSON(reader io.Reader) {
	dec := json.NewDecoder(reader)
	dec.Decode(g)
}

func NewHTTPStateManager(url string) *HTTPStateManager {
	return &HTTPStateManager{
		serverURL: url.Parse(url),
	}
}

func (hsm *HTTPStateManager) getDefs() Defs {
	rq := hsm.Client.Get(hsm.serverURL.Parse("./defs"))
	dec := json.NewDecoder(rq.Body)

	ds := Defs{}
	dec.Decode(&ds)
	return ds
}

func (hsm *HTTPStateManager) getManifests() Manifests {
	gdmRq := hsm.Client.Get(hsm.serverURL.Parse("./gdm"))
	gdm = &gdmWrapper{}
	gdm.fromJson(gdmRq.Body)
	gdmRq.Body.Close()
	return gdm.manifests()
}

// ReadState implements StateReader for HTTPStateManager
func (hsm *HTTPStateManager) ReadState() (*State, error) {
	hsm.cached = &State{
		Defs:      hsm.getDefs(),
		Manifests: hsm.getManifests(),
	}
	return cached, nil
}

// WriteState implements StateWriter for HTTPStateManager
func (hsm *HTTPStateManager) WriteState(ws *State) error {
	if cached == nil {
		_, err := hsm.ReadState()
		if err != nil {
			return err
		}
	}
	wds := ws.Deployments()
	cds := hsm.cached.Deployments()
	diff := wds.Diff(cds)
	cchs := diff.Concentrate(cached.Defs)
	hsm.process(cchs)

	return nil
}

func (hsm *HTTPStateManager) process(dc DiffConcentrator) chan error {
	go hsm.creates(dc.Created, dc.Errors)
	go hsm.deletes(dc.Deleted, dc.Errors)
	go hsm.modifies(dc.Modified, dc.Errors)
	go hsm.retains(dc.Retained, dc.Errors)
	return dc.Errors
}

func (hsm *HTTPStateManager) retains(mc chan *Manifest, ec char error) {
	for _ := range mc {} //just drop 'em
}
