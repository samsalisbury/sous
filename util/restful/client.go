package restful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"net/url"
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
		qparms       map[string]string
		body         io.Reader
		resourceJSON io.Reader
	}

	// HTTPClient interacts with a HTTPServer
	//   It's designed to handle basic CRUD operations in a safe and restful way.
	HTTPClient interface {
		Create(urlPath string, qParms map[string]string, rqBody interface{}, headers map[string]string) error
		Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, headers map[string]string) (UpdateDeleter, error)
	}

	// An Updater captures the state of a retrieved resource so that it can be updated later.
	Updater interface {
		Update(body Comparable, headers map[string]string) error
	}

	// A Deleter captures the state of a retrieved resource so that it can be later deleted.
	Deleter interface {
		Delete(headers map[string]string) error
	}

	// An UpdateDeleter allows for a given resource to be updated or deleted.
	UpdateDeleter interface {
		Updater
		Deleter
	}

	// DummyHTTPClient doesn't really make HTTP requests.
	DummyHTTPClient struct{}

	// Comparable is a required interface for Update and Delete, which provides
	// the mechanism for comparing the remote resource to the local data.
	Comparable interface {
		// EmptyReceiver should return a pointer to an "zero value" for the recieving type.
		// For example:
		///	//   func (x *X) EmptyReceiver() { return &X{} }
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

func (rs *resourceState) Update(qBody Comparable, headers map[string]string) error {
	return rs.client.update(rs.path, rs.qparms, rs, qBody, headers)
}

func (rs *resourceState) Delete(headers map[string]string) error {
	return rs.client.deelete(rs.path, rs.qparms, rs, headers)
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
	url, err := client.buildURL(urlPath, qParms)
	rq, err := client.buildRequest("GET", url, headers, nil, nil, err)
	rz, err := client.sendRequest(rq, err)
	state, err := client.getBody(rz, rzBody, err)
	if state != nil {
		state.client = client
		state.path = urlPath
		state.qparms = qParms
	}
	return state, errors.Wrapf(err, "Retrieve %s %v", urlPath, qParms)
}

// Create uses the contents of qBody to create a new resource at the server at urlPath/qParms
// It issues a PUT with "If-No-Match: *", so if a resource already exists, it'll return an error.
func (client *LiveHTTPClient) Create(urlPath string, qParms map[string]string, qBody interface{}, headers map[string]string) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		rq, err := client.buildRequest("PUT", url, addNoMatchStar(headers), nil, qBody, err)
		rz, err := client.sendRequest(rq, err)
		_, err = client.getBody(rz, nil, err)
		return err
	}(), "Create %s %v", urlPath, qParms)
}

func (client *LiveHTTPClient) deelete(urlPath string, qParms map[string]string, from *resourceState, headers map[string]string) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		etag := from.etag
		rq, err := client.buildRequest("DELETE", url, addIfMatch(headers, etag), nil, nil, err)
		rz, err := client.sendRequest(rq, err)
		_, err = client.getBody(rz, nil, err)
		return err
	}(), "Delete %s %v", urlPath, qParms)
}

func (client *LiveHTTPClient) update(urlPath string, qParms map[string]string, from *resourceState, qBody Comparable, headers map[string]string) error {
	return errors.Wrapf(func() error {
		url, err := client.buildURL(urlPath, qParms)
		//	etag := from.etag
		etag := from.etag
		rq, err := client.buildRequest("PUT", url, addIfMatch(headers, etag), from, qBody, err)
		rz, err := client.sendRequest(rq, err)
		_, err = client.getBody(rz, nil, err)
		return err
	}(), "Update %s %v", urlPath, qParms)
}

// Create implements HTTPClient on DummyHTTPClient - it does nothing and returns nil
func (*DummyHTTPClient) Create(urlPath string, qParms map[string]string, rqBody interface{}, headers map[string]string) error {
	return nil
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
	headers["If-None-Match"] = "*"
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

	//	client.Debugf("Sending %s %q", method, url)
	client.Debugf("Sending %s %q", method, url)
	JSON := &bytes.Buffer{}

	if rqBody != nil {
		JSON = encodeJSON(rqBody)
		if resource != nil {
			JSON = putbackJSON(resource.body, resource.resourceJSON, JSON)
		}
		client.Debugf("  body: %s", JSON.String())
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

func (client *LiveHTTPClient) sendRequest(rq *http.Request, ierr error) (*http.Response, error) {
	if ierr != nil {
		return nil, ierr
	}
	// needs to be fixed in coming log update
	client.Debugf("Sending %s %q", rq.Method, rq.URL)
	rz, err := client.httpRequest(rq)
	if err != nil {
		client.Debugf("Received %v", err)
		return rz, err
	}
	if rz != nil {
		client.Debugf("Received \"%s %s\" -> %d", rq.Method, rq.URL, rz.StatusCode)
	}
	return rz, err
}

func (client *LiveHTTPClient) getBody(rz *http.Response, rzBody interface{}, err error) (*resourceState, error) {
	if err != nil {
		return nil, err
	}
	defer rz.Body.Close()

	b, e := ioutil.ReadAll(rz.Body)
	if e != nil {
		client.Debugf("error reading from body: %v", e)
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
			resourceJSON: bytes.NewBuffer(rzJSON),
		}, errors.Wrapf(err, "processing response body")
	case rz.StatusCode < 200 || rz.StatusCode >= 300:
		return nil, errors.Errorf("%s: %#v", rz.Status, string(b))
	case rz.StatusCode == http.StatusConflict:
		return nil, errors.Wrap(retryableError(fmt.Sprintf("%s: %#v", rz.Status, string(b))), "getBody")
	}

}

func (client *LiveHTTPClient) logBody(dir, chName string, req *http.Request, b []byte, n int, err error) {
	reportServerMessage("logBody", chName, req, 0, int64(n), "", time.Duration(int64(0)), client.LogSink)
	comp := &bytes.Buffer{}
	if err := json.Compact(comp, b[0:n]); err != nil {
		reportServerMessage(fmt.Sprintf("%s", string(b)), chName, req, 0, int64(n), "", time.Duration(int64(0)), client.LogSink)
		reportServerMessage(fmt.Sprintf("problem compacting JSON for logging: %s)", err), chName, req, 0, int64(n), "", time.Duration(int64(0)), client.LogSink)
	} else {
		reportServerMessage(string(comp.String()), chName, req, 0, int64(n), "", time.Duration(int64(0)), client.LogSink)
	}
	reportServerMessage(fmt.Sprintf("%s %d bytes, result: %v", dir, n, err), chName, req, 0, int64(n), "", time.Duration(int64(0)), client.LogSink)
}

func (client *LiveHTTPClient) readerLogF(dir, chName string, req *http.Request) func(b []byte, n int, err error) {
	return func(b []byte, n int, err error) { client.logBody(dir, chName, req, b, n, err) }
}

func (client *LiveHTTPClient) httpRequest(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		reportServerMessage("Client -> <empty request body>", "", req, 0, int64(0), "", time.Duration(int64(0)), client.LogSink)
	} else {
		req.Body = readdebugger.New(req.Body, client.readerLogF("Sent", "Client ->", req))
	}
	rz, err := client.Client.Do(req)
	if rz == nil {
		return rz, err
	}
	if rz.Body == nil {
		reportServerMessage("Client <- <empty response body>", "", req, 0, int64(0), "", time.Duration(int64(0)), client.LogSink)
		return rz, err
	}

	rz.Body = readdebugger.New(rz.Body, client.readerLogF("Read", "Client <-", req))
	return rz, err
}

type clientMessage struct {
	logging.CallerInfo
	msg         string
	channelName string
	httpMsg     *messages.HTTPLogEntry
	isDebugMsg  bool
}

/* func reportClientMessage(msg string, channelName string, rz *http.Response, resName string, dur time.Duration, logger logging.LogSink) {
	// XXX dur should in fact be "start time.Time" and duration be computed here.
	// swaggering now depends on this, so it's more of a hassle.
	m := messages.BuildClientHTTPResponse(rz, resName, dur)
	m.ExcludeMe()
	reportMessage(m, msg, channelName, logger, true)
} */

// ReportServerHTTPResponse reports a response recieved by Sous as a client.
// n.b. this interface subject to change
func reportServerMessage(msg string, channelName string, rq *http.Request, statusCode int, contentLength int64, resName string, dur time.Duration, logger logging.LogSink) {
	m := messages.BuildServerHTTPResponse(rq, statusCode, contentLength, resName, dur)
	m.ExcludeMe()
	reportMessage(m, msg, channelName, logger, true)
}

func reportMessage(httpmsg *messages.HTTPLogEntry, msg string, channelName string, log logging.LogSink, debug ...bool) {
	debugStmt := false
	if len(debug) > 0 {
		debugStmt = debug[0]
	}

	msgLog := clientMessage{
		msg:         msg,
		CallerInfo:  logging.GetCallerInfo(logging.NotHere()),
		channelName: channelName,
		httpMsg:     httpmsg,
		isDebugMsg:  debugStmt,
	}
	logging.Deliver(msgLog, log)
}

func (msg clientMessage) DefaultLevel() logging.Level {
	level := logging.WarningLevel
	if msg.isDebugMsg {
		level = logging.DebugLevel
	}

	return level
}

func (msg clientMessage) Message() string {
	return msg.composeMsg()
}

func (msg clientMessage) composeMsg() string {
	return fmt.Sprintf("%s: channel name %s, status %d", msg.msg, msg.channelName, msg.httpMsg.Status())
}

func (msg clientMessage) EachField(f logging.FieldReportFn) {
	//f("@loglov3-otl", "sous-http-v1") //httpMsg for now will be adding the otl type, might need refactor
	f("channel_name", msg.channelName)
	msg.httpMsg.EachFieldWithoutCallerInfo(f)
	msg.CallerInfo.EachField(f)
}
