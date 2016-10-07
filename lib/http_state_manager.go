package sous

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
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

func (hsm *HTTPStateManager) retains(mc chan *Manifest, ec chan error) {
	for _ := range mc {
	} //just drop 'em
}

func (hsm *HTTPStateManager) manifestURL(m *Manifest) url.URL {
	murl := url.Parse("./manifests")
	mqry := url.Values{}
	mqry.Set("repo", m.Source.Repo)
	mqry.Set("offset", m.Source.Offset)
	mqry.Set("flavor", m.Flavor)
	murl.RawQuery = mqry.String()
	return hsm.serverURL.ResolveReference(murl)
}

func (hsm *HTTPStateManager) manifestJSON(m *Manifest) io.Reader {
	buf := bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(m)
	return buf
}

func (hsm *HTTPStateManager) jsonManifest(buf io.Reader) *Manifest {
	m := &Manifest{}
	dec := json.NewDecoder(buf)
	dec.Decode(m)
	return m
}

func (hsm *HTTPStateManager) creates(mc chan *Manifest, ec chan error) {
	for m := range mc {
		if err := hsm.create(m); err != nil {
			ec <- err
		}
	}
}
func (hsm *HTTPStateManager) deletes(mc chan *Manifest, ec chan error) {
	for m := range mc {
		if err := hsm.del(m); err != nil {
			ec <- err
		}
	}
}

func (hsm *HTTPStateManager) modifies(mc chan *ManifestPair, ec chan error) {
	for m := range mc {
		if err := hsm.modify(m); err != nil {
			ec <- err
		}
	}
}

func (hsm *HTTPStateManager) create(m *Manifest) error {
	rq := http.NewRequest("PUT", hsm.manifestURL(m), hsm.manifestJSON(m))
	rq.Header.Add("If-None-Match", "*")
	rz, err := hsm.Client.Do(rq)
	defer close(rz.Body)
	if err != nil {
		return err //XXX network problems? retry?
	}
	if rz.StatusCode != 200 {
		return errors.Errorf("%s: %#v", rz.Status, m)
	}
	return nil
}

func (hsm *HTTPStateManager) del(m *Manifest) error {
	grq := http.NewRequest("GET", hsm.manifestURL(m))
	grz, err := hsm.Client.Do(grq)
	defer close(grz.Body)
	if err != nil {
		return err
	}
	if !(grz.StatusCode >= 200 && grz.StatusCode < 300) {
		return errors.Errorf("%s: %#v", grz.Status, m)
	}
	rm := jsonManifest(grz.Body)
	if !rm.Equall(m) {
		return errors.Errorf("Remote and deleted manifests don't match: \n%#v\n%#v", rm, m)
	}
	etag := grz.Header.Get("Etag")
	drq := http.NewRequest("DELETE", hsm.manifestURL(m))
	drq.Header.Add("If-Match", etag)
	drz, err := hsm.Client.Do(drq)
	if err != nil {
		return err
	}
	if !(drz.StatusCode >= 200 && drz.StatusCode < 300) {
		return errors.Errorf("Delete failed: %s", drz.Status)
	}
	return nil
}

func (hsm *HTTPStateManager) modify(mp *ManifestPair) error {
	bf := mp.Prior
	af := mp.Post

	grq := http.NewRequest("GET", hsm.manifestURL(bf))
	grz, err := hsm.Client.Do(grq)
	defer close(grz.Body)
	if err != nil {
		return err
	}
	if !(grz.StatusCode >= 200 && grz.StatusCode < 300) {
		return errors.Errorf("%s: %#v", grz.Status, bf)
	}
	rm := jsonManifest(grz.Body)
	if !rm.Equall(bf) {
		return errors.Errorf("Remote and prior manifests don't match: \n%#v\n%#v", rm, bf)
	}
	etag := grz.Header.Get("Etag")

	prq := http.NewRequest("PUT", hsm.manifestURL(af), hsm.manifestJSON(af))
	prq.Header.Add("If-Match", etag)
	prz, err = hsm.Client.Do(prq)
	if !(prz.StatusCode >= 200 && prz.StatusCode < 300) {
		return errors.Errorf("Update failed: %s / %#v", prz.Status, af)
	}
}
