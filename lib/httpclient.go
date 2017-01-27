package sous

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type (
	// LiveHTTPClient interacts with a Sous http server.
	//   It's designed to handle basic CRUD operations in a safe and restful way.
	LiveHTTPClient struct {
		serverURL *url.URL
		http.Client
	}

	// HTTPClient interacts with a HTTPServer
	//   It's designed to handle basic CRUD operations in a safe and restful way.
	HTTPClient interface {
		Create(urlPath string, qParms map[string]string, rqBody interface{}) error
		Retrieve(urlPath string, qParms map[string]string, rzBody interface{}) error
		Update(urlPath string, qParms map[string]string, from, qBody Comparable) error
		Delete(urlPath string, qParms map[string]string, from Comparable) error
	}

	DummyHTTPClient struct{}

	// Comparable is a required interface for Update and Delete, which provides
	// the mechanism for comparing the remote resource to the local data.
	Comparable interface {
		// EmptyReceiver should return a pointer to an "zero value" for the recieving type.
		// For example:
		//   func (x *X) EmptyReceiver() { return &X{} }
		EmptyReceiver() Comparable

		// VariancesFrom returns a list of differences from another Comparable.
		// If the two structs are equivalent, it should return an empty list.
		// Usually, the first check will be for identical type, and return "types differ."
		VariancesFrom(Comparable) Variances
	}

	// Variances is a list of differences between two structs.
	Variances []string
)

// NewClient returns a new LiveHTTPClient for a particular serverURL.
func NewClient(serverURL string) (*LiveHTTPClient, error) {
	u, err := url.Parse(serverURL)

	client := &LiveHTTPClient{
		serverURL: u,
	}

	// XXX: This is in response to a mysterious issue surrounding automatic gzip
	// and Etagging The client receives a gzipped response with "--gzip" appended
	// to the original Etag The --gzip isn't stripped by whatever does it,
	// although the body is decompressed on the server side.  This is a hack to
	// address that issue, which should be resolved properly.
	client.Client.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment, DisableCompression: true}

	return client, errors.Wrapf(err, "new Sous REST client")
}

// ****

// Retrieve makes a GET request on urlPath, after transforming qParms into ?&=
// style query params. It deserializes the returned JSON into rzBody. Errors
// are returned if anything goes wrong, including a non-Success HTTP result
// (but note that there may be a response anyway.
func (client *LiveHTTPClient) Retrieve(urlPath string, qParms map[string]string, rzBody interface{}) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		rq, err := client.buildRequest("GET", url, nil, nil, err)
		rz, err := client.sendRequest(rq, err)
		return client.getBody(rz, rzBody, err)
	}(), "Retrieve %s", urlPath)
}

// Create uses the contents of qBody to create a new resource at the server at urlPath/qParms
// It issues a PUT with "If-No-Match: *", so if a resource already exists, it'll return an error.
func (client *LiveHTTPClient) Create(urlPath string, qParms map[string]string, qBody interface{}) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		rq, err := client.buildRequest("PUT", url, noMatchStar(), qBody, err)
		rz, err := client.sendRequest(rq, err)
		return client.getBody(rz, nil, err)
	}(), "Create %s", urlPath)
}

// Update changes the representation of a given resource.
// It compares the known value to from, and rejects if they're different (on
// the grounds that the client is going to clobber a value it doesn't know
// about.) Then it issues a PUT with "If-Match: <etag of from>" so that the
// server can check that we're changing from a known value.
func (client *LiveHTTPClient) Update(urlPath string, qParms map[string]string, from, qBody Comparable) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		etag, err := client.getBodyEtag(url, from, err)
		rq, err := client.buildRequest("PUT", url, ifMatch(etag), qBody, err)
		rz, err := client.sendRequest(rq, err)
		return client.getBody(rz, nil, err)
	}(), "Update %s", urlPath)
}

// Delete removes a resource from the server, granted that we know the resource that we're removing.
// It functions similarly to Update, but issues DELETE requests.
func (client *LiveHTTPClient) Delete(urlPath string, qParms map[string]string, from Comparable) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		etag, err := client.getBodyEtag(url, from, err)
		rq, err := client.buildRequest("DELETE", url, ifMatch(etag), nil, err)
		rz, err := client.sendRequest(rq, err)
		return client.getBody(rz, nil, err)
	}(), "Delete %s", urlPath)
}

// Create implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Create(urlPath string, qParms map[string]string, rqBody interface{}) error {
	return nil
}

// Retrieve implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Retrieve(urlPath string, qParms map[string]string, rzBody interface{}) error {
	return nil
}

// Update implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Update(urlPath string, qParms map[string]string, from, qBody Comparable) error {
	return nil
}

// Delete implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Delete(urlPath string, qParms map[string]string, from Comparable) error {
	return nil
}

// ***

func noMatchStar() map[string]string {
	return map[string]string{"If-None-Match": "*"}
}

func ifMatch(etag string) map[string]string {
	return map[string]string{"If-Match": etag}
}

// ****

func (client *LiveHTTPClient) buildURL(urlPath string, qParms map[string]string) (urlS string, err error) {
	URL, err := client.serverURL.Parse(urlPath)
	if err != nil {
		return
	}
	if qParms == nil {
		return URL.String(), nil
	}
	qry := url.Values{}
	for k, v := range qParms {
		qry.Set(k, v)
	}
	URL.RawQuery = qry.Encode()
	return client.serverURL.ResolveReference(URL).String(), nil
}

func (client *LiveHTTPClient) getBodyEtag(url string, body Comparable, ierr error) (etag string, err error) {
	if ierr != nil {
		err = ierr
		return
	}
	Log.Debug.Printf("Getting existing resource from %s", url)

	rzBody := body.EmptyReceiver()

	var rq *http.Request
	var rz *http.Response
	err = errors.Wrapf(func() error {
		rq, err = client.buildRequest("GET", url, nil, nil, nil)
		rz, err = client.sendRequest(rq, err)
		return client.getBody(rz, rzBody, err)
	}(), "etag for %s", url)
	if err != nil {
		return
	}

	differences := rzBody.VariancesFrom(body)
	if len(differences) > 0 {
		return "", errors.Errorf("Remote and local versions of %s resource don't match: %#v", url, differences)
	}
	return rz.Header.Get("Etag"), nil
}

func (client *LiveHTTPClient) buildRequest(method, url string, headers map[string]string, rqBody interface{}, ierr error) (*http.Request, error) {
	if ierr != nil {
		return nil, ierr
	}

	Log.Debug.Printf("Sending %s %q", method, url)

	var JSON io.Reader

	if rqBody != nil {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.Encode(rqBody)
		Log.Debug.Printf("  body: %s", buf.String())
		JSON = buf
	}

	rq, err := http.NewRequest(method, url, JSON)

	if headers != nil {
		for k, v := range headers {
			rq.Header.Add(k, v)
		}
	}

	return rq, err
}

func (client *LiveHTTPClient) sendRequest(rq *http.Request, ierr error) (*http.Response, error) {
	if ierr != nil {
		return nil, ierr
	}
	rz, err := client.httpRequest(rq)
	if err != nil {
		Log.Debug.Printf("Received %v", err)
		return rz, err
	}
	if rz != nil {
		Log.Debug.Printf("Received \"%s %s\" -> %d", rq.Method, rq.URL, rz.StatusCode)
	}
	return rz, err
}

func (client *LiveHTTPClient) getBody(rz *http.Response, rzBody interface{}, err error) error {
	if err != nil {
		return err
	}
	defer rz.Body.Close()

	if rzBody != nil {
		dec := json.NewDecoder(rz.Body)
		err = dec.Decode(rzBody)
	}

	if rz.StatusCode < 200 || rz.StatusCode >= 300 {
		b, e := ioutil.ReadAll(rz.Body)
		if e != nil {
			b = []byte{}
		}
		return errors.Errorf("%s: %#v", rz.Status, string(b))
	}
	return errors.Wrapf(err, "processing response body")
}

func logBody(dir, chName string, req *http.Request, b []byte, n int, err error) {
	Log.Vomit.Printf("%s %s %q", chName, req.Method, req.URL)
	comp := &bytes.Buffer{}
	if err := json.Compact(comp, b[0:n]); err != nil {
		Log.Vomit.Print(string(b))
		Log.Vomit.Printf("(problem compacting JSON for logging: %s)", err)
	} else {
		Log.Vomit.Print(comp.String())
	}
	Log.Vomit.Printf("%s %d bytes, result: %v", dir, n, err)
}

func (client *LiveHTTPClient) httpRequest(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		Log.Vomit.Printf("Client -> %s %q <empty request body>", req.Method, req.URL)
	} else {
		req.Body = NewReadDebugger(req.Body, func(b []byte, n int, err error) {
			logBody("Sent", "Client ->", req, b, n, err)
		})
	}
	rz, err := client.Client.Do(req)
	if rz == nil {
		return rz, err
	}
	if rz.Body == nil {
		Log.Vomit.Printf("Client <- %s %q %d <empty response body>", req.Method, req.URL, rz.StatusCode)
		return rz, err
	}

	rz.Body = NewReadDebugger(rz.Body, func(b []byte, n int, err error) {
		logBody("Read", "Client <-", req, b, n, err)
	})
	return rz, err
}
