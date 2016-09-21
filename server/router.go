package server

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime/debug"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

type (
	// A Exchanger has an Exchange method - which is presumed to write to an
	// injected ResponseWriter
	Exchanger interface {
		Exchange() (interface{}, int)
	}

	// The metahandler collects common behavior for route handlers
	MetaHandler struct {
		router   *httprouter.Router
		graphFac GraphFactory
	}

	// A PanicHandler processes panics into 500s
	PanicHandler struct {
		*sous.LogSet
	}

	// ResponseWriter wraps the the http.ResponseWriter interface
	// XXX This is a workaround for Psyringe
	ResponseWriter struct {
		http.ResponseWriter
	}

	// An ExchangeFactory builds an Exchanger -
	// they're used to configure the RouteMap
	ExchangeFactory func() Exchanger

	// A GraphFactory builds a SousGraph
	GraphFactory func() *graph.SousGraph

	// QueryValues wrap url.Values to keep them needing to be re-exported
	QueryValues struct {
		url.Values
	}

	routeEntry struct {
		Name, Method, Path string
		Exchange           ExchangeFactory
	}

	// EmptyReader is an empty reader - returns EOF immediately
	EmptyReader struct{}

	// RouteMap is a list of entries for routing
	RouteMap []routeEntry
)

var (
	// SousRouteMap is the configuration of route for the application
	SousRouteMap = RouteMap{
		{"gdm", "GET", "/gdm", NewGDMHandler},
	}
)

// BuildRouter builds a returns an http.Handler based on some constant configuration
func BuildRouter(rm RouteMap, grf func() *graph.SousGraph) http.Handler {
	r := httprouter.New()
	mh := &MetaHandler{
		graphFac: grf,
		router:   r,
	}
	mh.InstallPanicHandler()

	for _, e := range rm {
		m := e.Method
		switch strings.ToUpper(m) {
		case "GET":
			r.Handle("GET", e.Path, mh.GetHandling(e.Exchange))
			r.Handle("HEAD", e.Path, mh.HeadHandling(e.Exchange))
		case "HEAD":
			r.Handle("HEAD", e.Path, mh.HeadHandling(e.Exchange))
		case "POST":
			r.Handle("POST", e.Path, mh.PostHandling(e.Exchange))
		case "PUT":
			r.Handle("PUT", e.Path, mh.PutHandling(e.Exchange))
		case "DELETE":
			r.Handle("DELETE", e.Path, mh.DeleteHandling(e.Exchange))
		default:
			r.Handle(m, e.Path, mh.OtherHandling(e.Exchange))
		}
	}

	return r
}

// PathFor constructs a URL which should route back to the named route, with
// supplied parameters
func (rm *RouteMap) PathFor(name string, params map[string]string) (string, error) {
	for _, e := range *rm {
		if e.Name == name {
			// TODO actually handle params
			return e.Path, nil
		}
	}
	return "", errors.Errorf("No route found for name %q", name)
}

// Handle recovers from panics, returns a 500 and logs the error
// It uses the LogSet provided by the graph
func (ph *PanicHandler) Handle(w http.ResponseWriter, r *http.Request, recovered interface{}) {
	if r := recover(); r != nil {
		w.WriteHeader(http.StatusInternalServerError)
		ph.LogSet.Warn.Printf("%+v", recovered)
		ph.LogSet.Warn.Print(string(debug.Stack()))
		ph.LogSet.Warn.Print("Recovered, returned 500")
		// XXX in a dev mode, print the panic in the response body
		// (normal ops it might leak secure data)
	}
}

// InstallPanicHandler installs an panic handler into the router
func (mh *MetaHandler) InstallPanicHandler() {
	g := graphFac()
	ph := &PanicHandler{}
	g.Inject(ph)
	r.PanicHandler = func(w http.ResponseWriter, r *http.Request, recovered interface{}) {
		ph.Handle(w, r, recovered)
	}

}

func parseQueryValues(req *http.Request) (*QueryValues, error) {
	v, err := url.ParseQuery(req.URL.RawQuery)
	return &QueryValues{v}, err
}

func (mh *MetaHandler) exchangeGraph(w http.ResponseWriter, r *http.Request, p httprouter.Params) *graph.SousGraph {
	g := mh.graphFac()
	g.Add(&ResponseWriter{ResponseWriter: w}, r, p)
	g.Add(parseQueryValues)
	return g
}

func (mh *MetaHandler) injectedHandler(factory ExchangeFactory, w http.ResponseWriter, r *http.Request, p httprouter.Params) Exchanger {
	h := factory()

	mh.exchangeGraph(w, r, p).Inject(h)

	return h
}

func (mh *MetaHandler) writeHeaders(status int, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(status)
}

func (mh *MetaHandler) renderData(data interface{}, status int, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	mh.writeHeaders(status, w, r)
	e := json.NewEncoder(w)
	e.Encode(data)
}

func (mh *MetaHandler) synthResponse(req *http.Request) *http.Response {
	gw := httptest.NewRecorder()
	mh.router.ServeHTTP(gw, gr)
	return gw.Response()
}

func (*EmptyReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func emptyBody() io.ReadCloser {
	return ioutil.NopCloser(&EmptyReader)
}

func copyRequest(req *http.Request) *http.Request {
	new := &http.Request{}
	*new = *req
	if req.URL != nil {
		new.URL = &url.URL
		*new.URL = *req.URL
	}
	new.Body = emptyBody() //users must copy body themselves
	return new
}

// GetHandling (sometimes) updates the local copy of the GDM and formats it
func (mh *MetaHandler) GetHandling(graphFac func() *graph.SousGraph, factory ExchangeFactory) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		h := mh.injectedHandler(factory, w, r, p)
		data, status := h.Exchange()
		mh.renderData(data, status, w, r)
	}
}

// HeadHandling (sometimes) updates the local copy of the GDM and formats it
func (mh *MetaHandler) HeadHandling(graphFac func() *graph.SousGraph, factory ExchangeFactory) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		h := mh.injectedHandler(factory, w, r, p)
		data, status := h.Exchange()
		mh.writeHeaders(status, w, r)
	}
}

// PutHandling handles PUT requests
func (mh *MetaHandler) PutHandling(graphFac func() *graph.SousGraph, factory ExchangeFactory) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if r.Header.Get("If-Match") == "" && r.Header.Get("In-None-Match") == "" {
			w.WriteHeader(http.StatusPreconditionRequired)
			return
		}

		gr := copyRequest(req)
		gr.Method = "GET"
		grez := mh.synthResponse(gr)

		if r.Header.Get("If-None-Match") == "*" && grez.StatusCode != 404 {
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}
		if etag := r.Header.Get("If-Match"); etag != "" {
			if mh.EtagFor(grez.Body) != etag {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
		}
		h := mh.injectedHandler(factory, w, r, p)
		data, status := h.Exchange()
		mh.renderData(data, status, w, r)
	}
}

// The irony here is that there's very little the MetaHandler can do automatically for the Handler, so DELETE and POST behave like GET.

// OtherHandling handles requests for methods other than the known
var OtherHandling = GetHandling

// DeleteHandling handles DELETE requests
var DeleteHandling = GetHandling

// PostHandling handles POST requests
var PostHandling = GetHandling
