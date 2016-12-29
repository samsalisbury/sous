package sous

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type (
	// An HTTPStateManager gets state from a Sous server and transmits updates
	// back to that server.
	HTTPStateManager struct {
		serverURL *url.URL
		cached    *State
		http.Client
	}

	gdmWrapper struct {
		Deployments []*Deployment
	}

	// ReadDebugger wraps a ReadCloser and logs the data as it buffers past.
	ReadDebugger struct {
		wrapped io.ReadCloser
		logged  bool
		count   int
		read    []byte
		log     func([]byte, int, error)
	}
)

// NewReadDebugger creates a new ReadDebugger that wraps a ReadCloser
func NewReadDebugger(rc io.ReadCloser, log func([]byte, int, error)) *ReadDebugger {
	return &ReadDebugger{
		wrapped: rc,
		read:    []byte{},
		log:     log,
	}
}

// Read implements Reader on ReadDebugger
func (rd *ReadDebugger) Read(p []byte) (int, error) {
	n, err := rd.wrapped.Read(p)
	rd.read = append(rd.read, p...)
	rd.count += n
	if err != nil {
		rd.log(rd.read, rd.count, err)
		rd.logged = true
	}
	return n, err
}

// Close implements Closer on ReadDebugger
func (rd *ReadDebugger) Close() error {
	err := rd.wrapped.Close()
	if !rd.logged {
		rd.log(rd.read, rd.count, err)
		rd.logged = true
	}
	return err
}

func (g *gdmWrapper) manifests(defs Defs) (Manifests, error) {
	ds := NewDeployments()
	for _, d := range g.Deployments {
		ds.Add(d)
	}
	return ds.Manifests(defs)
}

func (g *gdmWrapper) fromJSON(reader io.Reader) {
	dec := json.NewDecoder(reader)
	dec.Decode(g)
}

// NewHTTPStateManager creates a new HTTPStateManager.
func NewHTTPStateManager(us string) (*HTTPStateManager, error) {
	u, err := url.Parse(us)

	hsm := &HTTPStateManager{
		serverURL: u,
	}

	// XXX: This is in response to a mysterios issue surrounding automatic gzip and
	// Etagging The client receives a gzipped response with "--gzip" appended to
	// the original Etag The --gzip isn't stripped by whatever does it, although
	// the body is decompressed on the server side.
	// This is a hack to address that issue, which should be resolved properly
	hsm.Client.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment, DisableCompression: true}

	return hsm, errors.Wrapf(err, "new state manager")
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
func (hsm *HTTPStateManager) WriteState(s *State) error {
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
	cchs := diff.Concentrate(s.Defs)
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

func (hsm *HTTPStateManager) manifestURL(m *Manifest) (string, error) {
	murl, err := url.Parse("./manifest")
	if err != nil {
		return "", err
	}
	mqry := url.Values{}
	mqry.Set("repo", m.Source.Repo)
	mqry.Set("offset", m.Source.Dir)
	mqry.Set("flavor", m.Flavor)
	murl.RawQuery = mqry.Encode()
	return hsm.serverURL.ResolveReference(murl).String(), nil
}

func (hsm *HTTPStateManager) manifestJSON(m *Manifest) io.Reader {
	buf := &bytes.Buffer{}
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

func (hsm *HTTPStateManager) getDefs() (Defs, error) {
	ds := Defs{}
	url, err := hsm.serverURL.Parse("./defs")
	if err != nil {
		return ds, errors.Wrapf(err, "getting defs")
	}

	Log.Debug.Printf("Reading definitions from %s", url)

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return ds, errors.Wrapf(err, "getting defs")
	}
	rz, err := hsm.httpRequest(req)
	if err != nil {
		return ds, errors.Wrapf(err, "getting defs")
	}
	defer rz.Body.Close()

	dec := json.NewDecoder(rz.Body)

	return ds, errors.Wrapf(dec.Decode(&ds), "getting defs")
}

func (hsm *HTTPStateManager) getManifests(defs Defs) (Manifests, error) {
	url, err := hsm.serverURL.Parse("./gdm")
	if err != nil {
		return Manifests{}, errors.Wrapf(err, "getting manifests")
	}

	Log.Debug.Printf("Reading manifests from %s", url)

	rq, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return Manifests{}, errors.Wrapf(err, "getting manifests")
	}
	gdmRz, err := hsm.httpRequest(rq)
	if err != nil {
		return Manifests{}, errors.Wrapf(err, "getting manifests")
	}
	defer gdmRz.Body.Close()
	gdm := &gdmWrapper{}
	gdm.fromJSON(gdmRz.Body)
	return gdm.manifests(defs)
}

func (hsm *HTTPStateManager) httpRequest(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		Log.Vomit.Printf("-> %s %q", req.Method, req.URL)
	} else {
		req.Body = NewReadDebugger(req.Body, func(b []byte, n int, err error) {
			Log.Vomit.Printf("-> %s %q:\n%sSent %d bytes, result: %v", req.Method, req.URL, string(b), n, err)
		})
	}
	rz, err := hsm.Client.Do(req)
	log.Print(err)
	if rz == nil {
		return rz, err
	}
	if rz.Body == nil {
		Log.Vomit.Printf("<- %s %q %d", req.Method, req.URL, rz.StatusCode)
	} else {
		rz.Body = NewReadDebugger(rz.Body, func(b []byte, n int, err error) {
			Log.Vomit.Printf("<- %s %q %d:\n%sRead %d bytes, result: %v", req.Method, req.URL, rz.StatusCode, string(b), n, err)
		})
	}
	return rz, err
}

func (hsm *HTTPStateManager) create(m *Manifest) error {
	murl, err := hsm.manifestURL(m)
	if err != nil {
		return err
	}
	json := hsm.manifestJSON(m)
	Log.Debug.Printf("Creating manifest: PUT %q", murl)
	Log.Debug.Printf("Creating manifest: %s", json)
	rq, err := http.NewRequest("PUT", murl, json)
	if err != nil {
		return errors.Wrapf(err, "create manifest request")
	}
	rq.Header.Add("If-None-Match", "*")
	rz, err := hsm.httpRequest(rq)
	if err != nil {
		return err //XXX network problems? retry?
	}
	defer rz.Body.Close()
	if rz.StatusCode != 200 {
		return errors.Errorf("%s: %#v", rz.Status, m)
	}
	return nil
}

func (hsm *HTTPStateManager) del(m *Manifest) error {
	u, etag, err := hsm.getManifestEtag(m)
	if err != nil {
		return errors.Wrapf(err, "delete manifest request")
	}

	drq, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return errors.Wrapf(err, "delete manifest request")
	}
	drq.Header.Add("If-Match", etag)
	drz, err := hsm.httpRequest(drq)
	if err != nil {
		return errors.Wrapf(err, "delete manifest request")
	}
	defer drz.Body.Close()
	if !(drz.StatusCode >= 200 && drz.StatusCode < 300) {
		return errors.Errorf("Delete %s failed: %s", u, drz.Status)
	}
	return nil
}

func (hsm *HTTPStateManager) modify(mp *ManifestPair) error {
	bf := mp.Prior
	af := mp.Post
	u, etag, err := hsm.getManifestEtag(bf)
	if err != nil {
		return errors.Wrapf(err, "modify request")
	}

	// XXX I don't think the URL should be *able* to be different here.
	u, err = hsm.manifestURL(af)
	if err != nil {
		return err
	}

	json := hsm.manifestJSON(af)
	Log.Debug.Printf("Updating manifest at %q", u)
	Log.Debug.Printf("Updating manifest to %s", json)

	prq, err := http.NewRequest("PUT", u, json)
	if err != nil {
		return errors.Wrapf(err, "modify request")
	}
	prq.Header.Add("If-Match", etag)
	prz, err := hsm.httpRequest(prq)
	if err != nil {
		return errors.Wrapf(err, "modify request")
	}
	defer prz.Body.Close()
	if !(prz.StatusCode >= 200 && prz.StatusCode < 300) {
		return errors.Errorf("Update failed: %s / %#v", prz.Status, af)
	}
	return nil
}

func (hsm *HTTPStateManager) getManifestEtag(m *Manifest) (string, string, error) {
	u, err := hsm.manifestURL(m)
	if err != nil {
		return "", "", err
	}

	Log.Debug.Printf("Getting manifest from %s", u)

	grq, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", "", err
	}
	grz, err := hsm.httpRequest(grq)
	if err != nil {
		return "", "", err
	}
	defer grz.Body.Close()
	if !(grz.StatusCode >= 200 && grz.StatusCode < 300) {
		return "", "", errors.Errorf("GET %s, %s: %#v", u, grz.Status, m)
	}
	rm := hsm.jsonManifest(grz.Body)
	different, differences := rm.Diff(m)
	if different {
		return "", "", errors.Errorf("Remote and local versions of manifest don't match: %#v", differences)
	}
	return u, grz.Header.Get("Etag"), nil
}
