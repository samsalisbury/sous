package restful

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/hydrogen18/memlistener"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/readdebugger"
	"github.com/pkg/errors"
)

type (
	// LiveHTTPClient interacts with a Sous http server.
	//   It's designed to handle basic CRUD operations in a safe and restful way.
	LiveHTTPClient struct {
		serverURL *url.URL
		http.Client
		logging.LogSink
		commonHeaders http.Header
	}

	resourceState struct {
		client       *LiveHTTPClient
		path, etag   string
		headers      http.Header
		qparms       map[string]string
		body         io.Reader
		resourceJSON io.Reader
	}

	// HTTPClient interacts with a HTTPServer
	//   It's designed to handle basic CRUD operations in a safe and restful way.
	HTTPClient interface {
		Create(urlPath string, qParms map[string]string, rqBody interface{}, headers map[string]string) (UpdateDeleter, error)
		Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, headers map[string]string) (UpdateDeleter, error)
	}

	// HTTPClientWithContext interacts with a HTTPServer
	//   It's designed to handle basic CRUD operations in a safe and restful way, adding context.
	HTTPClientWithContext interface {
		CreateCtx(ctx context.Context, urlPath string, qParms map[string]string, rqBody interface{}, headers map[string]string) (UpdateDeleter, error)
		RetrieveCtx(ctx context.Context, ctrlpath string, qparms map[string]string, rzbody interface{}, headers map[string]string) (UpdateDeleter, error)
	}

	// An Updater captures the state of a retrieved resource so that it can be updated later.
	Updater interface {
		Update(body Comparable, headers map[string]string) (UpdateDeleter, error)
	}

	// A Deleter captures the state of a retrieved resource so that it can be later deleted.
	Deleter interface {
		Delete(headers map[string]string) error
	}

	// An UpdateDeleter allows for a given resource to be updated or deleted.
	UpdateDeleter interface {
		Updater
		Deleter
		Location() string
	}

	// DummyHTTPClient doesn't really make HTTP requests.
	DummyHTTPClient struct{}

	// Comparable is a required interface for Update and Delete, which provides
	// the mechanism for comparing the remote resource to the local data.
	Comparable interface {
		// EmptyReceiver should return a pointer to an "zero value" for the recieving type.
		// For example:
		//   func (x *X) EmptyReceiver() Comparable { return &X{} }
		EmptyReceiver() Comparable

		// VariancesFrom returns a list of differences from another Comparable.
		// If the two structs are equivalent, it should return an empty list.
		// Usually, the first check will be for identical type, and return "types differ."
		VariancesFrom(Comparable) Variances
	}

	// Variances is a list of differences between two structs.
	Variances []string

	retryableError string
)

func (rs *resourceState) Update(qBody Comparable, headers map[string]string) (UpdateDeleter, error) {
	return rs.client.update(rs.path, rs.qparms, rs, qBody, headers)
}

func (rs *resourceState) Delete(headers map[string]string) error {
	return rs.client.delete(rs.path, rs.qparms, rs, headers)
}

func (rs *resourceState) Location() string {
	return rs.headers.Get("Location")
}

func (re retryableError) Error() string {
	return string(re)
}

// Retryable is a predicate on error that returns true if the error indicates
// that a subsequent attempt at e.g. an Update might succeed.
func Retryable(err error) bool {
	_, is := errors.Cause(err).(retryableError)
	return is
}

// NewClient returns a new LiveHTTPClient for a particular serverURL.
func NewClient(serverURL string, ls logging.LogSink, headers ...map[string]string) (*LiveHTTPClient, error) {
	u, err := url.Parse(serverURL)

	client := &LiveHTTPClient{
		serverURL:     u,
		LogSink:       ls,
		commonHeaders: buildHeaders(headers),
	}

	// XXX: This is in response to a mysterious issue surrounding automatic gzip
	// and Etagging The client receives a gzipped response with "--gzip" appended
	// to the original Etag The --gzip isn't stripped by whatever does it,
	// although the body is decompressed on the server side.  This is a hack to
	// address that issue, which should be resolved properly.
	client.Client.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment, DisableCompression: true}

	return client, errors.Wrapf(err, "new Sous REST client")
}

// NewInMemoryClient wraps a MemoryListener in a restful.Client
func NewInMemoryClient(handler http.Handler, ls logging.LogSink, headers ...map[string]string) (HTTPClient, error) {
	u, err := url.Parse("http://in.memory.server")
	if err != nil {
		return nil, err
	}

	ms := memlistener.NewInMemoryServer(handler)

	client := &LiveHTTPClient{
		serverURL:     u,
		LogSink:       ls,
		Client:        *ms.NewClient(),
		commonHeaders: buildHeaders(headers),
	}

	return client, nil
}

func buildHeaders(maybeHeaders []map[string]string) http.Header {
	hs := make(http.Header)
	if len(maybeHeaders) > 0 {
		for k, v := range maybeHeaders[0] {
			hs.Set(k, v)
		}
	}
	return hs
}

// ****

// Retrieve makes a GET request on urlPath, after transforming qParms into ?&=
// style query params. It deserializes the returned JSON into rzBody. Errors
// are returned if anything goes wrong, including a non-Success HTTP result
// (but note that there may be a response anyway.
// It returns an Updater so that the resource can be updated later
func (client *LiveHTTPClient) Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, headers map[string]string) (UpdateDeleter, error) {
	return client.RetrieveCtx(context.TODO(), urlPath, qParms, rzBody, headers)
}

//RetrieveCtx Add context to Retrieve to ability to cancel
func (client *LiveHTTPClient) RetrieveCtx(ctx context.Context, urlPath string, qParms map[string]string, rzBody interface{}, headers map[string]string) (UpdateDeleter, error) {
	rq, err := client.constructRequest(ctx, "GET", urlPath, qParms, nil, headers)
	rz, err := client.sendRequest(rq, err)
	state, err := client.extractBody(rz, rzBody, err)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %s", rq.URL, err)
	}
	return client.enrichState(state, urlPath, qParms), nil //errors.Wrapf(err, "Retrieve %s params: %v", urlPath, qParms)
}

// Create uses the contents of qBody to create a new resource at the server at urlPath/qParms
// It issues a PUT with "If-No-Match: *", so if a resource already exists, it'll return an error.
func (client *LiveHTTPClient) Create(urlPath string, qParms map[string]string, qBody interface{}, headers map[string]string) (UpdateDeleter, error) {
	return client.CreateCtx(context.TODO(), urlPath, qParms, qBody, headers)
}

//CreateCtx Add context to Create to ability to cancel
func (client *LiveHTTPClient) CreateCtx(ctx context.Context, urlPath string, qParms map[string]string, qBody interface{}, headers map[string]string) (UpdateDeleter, error) {
	rq, err := client.constructRequest(ctx, "PUT", urlPath, qParms, qBody, addNoMatchStar(headers))
	rz, err := client.sendRequest(rq, err)
	state, err := client.extractBody(rz, nil, err)
	return client.enrichState(state, urlPath, qParms), errors.Wrapf(err, "Create %s params: %v", urlPath, qParms)
}

func (client *LiveHTTPClient) enrichState(state *resourceState, urlPath string, qParms map[string]string) *resourceState {
	if state != nil {
		state.client = client
		state.path = urlPath
		state.qparms = qParms
	}
	return state
}

func (client *LiveHTTPClient) delete(urlPath string, qParms map[string]string, from *resourceState, headers map[string]string) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		etag := from.etag
		rq, err := client.buildRequest("DELETE", url, addIfMatch(headers, etag), nil, nil, err)
		rz, err := client.sendRequest(rq, err)
		_, err = client.extractBody(rz, nil, err)
		return err
	}(), "Delete %s params: %v", urlPath, qParms)
}

func (client *LiveHTTPClient) update(urlPath string, qParms map[string]string, from *resourceState, qBody Comparable, headers map[string]string) (UpdateDeleter, error) {
	state := new(resourceState)
	err := errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		etag := from.etag
		rq, err := client.buildRequest("PUT", url, addIfMatch(headers, etag), from, qBody, err)
		rz, err := client.sendRequest(rq, err)
		state, err = client.extractBody(rz, nil, err)
		client.enrichState(state, urlPath, qParms)
		if state != nil {
			state.headers = rz.Header
		}
		return err
	}(), "Update %s params: %v", urlPath, qParms)
	return state, err
}

// Create implements HTTPClient on DummyHTTPClient - it does nothing and returns nil.
func (*DummyHTTPClient) Create(urlPath string, qParms map[string]string, rqBody interface{}, headers map[string]string) (UpdateDeleter, error) {
	return nil, nil
}

// Retrieve implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, headers map[string]string) (UpdateDeleter, error) {
	return nil, nil
}

// Update implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) update(urlPath string, qParms map[string]string, from *resourceState, qBody Comparable, headers map[string]string) error {
	return nil
}

// Delete implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) deelete(urlPath string, qParms map[string]string, from *resourceState, headers map[string]string) error {
	return nil
}

// ***

func addNoMatchStar(headers map[string]string) map[string]string {
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["If-None-Match"]; !ok {
		headers["If-None-Match"] = "*"
	}
	return headers
}

func addIfMatch(headers map[string]string, etag string) map[string]string {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["If-Match"] = etag
	return headers
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

func (client *LiveHTTPClient) buildRequest(method, url string, headers map[string]string, resource *resourceState, rqBody interface{}, ierr error) (*http.Request, error) {
	if ierr != nil {
		return nil, ierr
	}

	JSON := &bytes.Buffer{}

	if rqBody != nil {
		JSON = encodeJSON(rqBody)
		if resource != nil {
			var err error
			JSON, err = putbackJSON(resource.body, resource.resourceJSON, JSON)
			if err != nil {
				return nil, err
			}
		}
		messages.ReportLogFieldsMessage("JSON Body", logging.DebugLevel, client.LogSink, JSON.String())
	}

	rq, err := http.NewRequest(method, url, JSON)

	if headers == nil {
		headers = map[string]string{}
	}

	client.updateHeaders(rq, headers)

	return rq, err
}

func (client *LiveHTTPClient) updateHeaders(rq *http.Request, headers map[string]string) {
	for k, v := range headers {
		rq.Header.Add(k, v)
	}

	for k, v := range client.commonHeaders {
		if _, overridden := rq.Header[textproto.CanonicalMIMEHeaderKey(k)]; !overridden {
			for _, s := range v {
				rq.Header.Add(k, s)
			}
		}
	}
}

func (client *LiveHTTPClient) constructRequest(ctx context.Context, method string, urlPath string, qParms map[string]string, qBody interface{}, headers map[string]string) (*http.Request, error) {
	url, err := client.buildURL(urlPath, qParms)
	rq, err := client.buildRequest(method, url, headers, nil, qBody, err)
	if ctx != nil {
		rq = rq.WithContext(ctx)
	}
	return rq, err
}

func (client *LiveHTTPClient) sendRequest(rq *http.Request, ierr error) (*http.Response, error) {
	if ierr != nil {
		return nil, ierr
	}
	// needs to be fixed in coming log update
	rz, err := client.performHTTPRequest(rq)
	if err != nil {
		return rz, err
	}
	return rz, err
}

func checkContentType(ct string) error {
	switch {
	default:
		return fmt.Errorf("bad content type %q", ct)
	case ct == "", ct == "application/json",
		strings.HasPrefix(ct, "text/plain"):
		return nil
	}
}

func (client *LiveHTTPClient) extractBody(rz *http.Response, rzBody interface{}, err error) (*resourceState, error) {
	if err != nil {
		return nil, err
	}
	defer rz.Body.Close()

	if err := checkContentType(rz.Header.Get("Content-Type")); err != nil {
		return nil, fmt.Errorf("%s (status %s)", err, rz.Status)
	}

	b, e := ioutil.ReadAll(rz.Body)
	if e != nil {
		logging.ReportError(client.LogSink, errors.Wrap(e, "error reading from body"))
		b = []byte{}
	}

	if rzBody != nil {
		err = json.Unmarshal(b, rzBody)
	}

	switch {
	default:
		rzJSON, merr := json.Marshal(rzBody)
		if err == nil {
			err = merr
		}
		return &resourceState{
			etag:         rz.Header.Get("ETag"),
			body:         bytes.NewBuffer(b),
			headers:      rz.Header,
			resourceJSON: bytes.NewBuffer(rzJSON),
		}, errors.Wrapf(err, "processing response body")
	case rz.StatusCode < 200 || rz.StatusCode >= 300:
		return nil, errors.Errorf("%s: %s", rz.Status, string(b))
	case rz.StatusCode == http.StatusConflict:
		return nil, errors.Wrap(retryableError(fmt.Sprintf("%s: %#v", rz.Status, string(b))), "getBody")
	case rz.Header.Get("Content-Type") != "application/json" && len(b) > 0:
		return nil, errors.Errorf("%s: Not JSON response: %q\n'%s'", rz.Status, rz.Header.Get("Content-Type"), string(b))
	}

}

func bodyMessage(b []byte, n int, err error) string {
	comp := &bytes.Buffer{}
	if err == io.EOF {
		err = nil
	}
	if cerr := json.Compact(comp, b[0:n]); cerr != nil {
		return fmt.Sprintf("body: %d bytes: %q (read err: %v)", n, string(b), err)
	}
	return fmt.Sprintf("body: %d bytes, %s (read err: %v)", n, comp.String(), err)
}

func (client *LiveHTTPClient) performHTTPRequest(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		messages.ReportClientHTTPRequest(client.LogSink, "<empty request body>", req, "")
	} else {
		req.Body = readdebugger.New(req.Body, func(b []byte, n int, err error) {
			messages.ReportClientHTTPRequest(client.LogSink, bodyMessage(b, n, err), req, "")
		})
	}

	sendTime := time.Now()

	rz, err := client.Client.Do(req)
	recvTime := time.Now()
	rqDur := recvTime.Sub(sendTime)

	if err != nil {
		logging.ReportError(client.LogSink, errors.Wrapf(err, "error performing HTTP request %s %q", req.Method, req.URL.String()))
	}
	if rz == nil {
		return rz, err
	}
	if rz.Body == nil {
		messages.ReportClientHTTPResponse(client.LogSink, "<empty response body>", rz, "placeholder resource name", rqDur)
		return rz, err
	}

	rz.Body = readdebugger.New(rz.Body, func(b []byte, n int, err error) {
		messages.ReportClientHTTPResponse(client.LogSink, bodyMessage(b, n, err), rz, "", rqDur)
	})

	return rz, err
}
