package restful

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/util/logging"
)

type (
	// A Exchanger has an Exchange method - which is presumed to write to an
	// injected ResponseWriter
	Exchanger interface {
		Exchange() (interface{}, int)
	}

	// An ExchangeFactory builds an Exchanger -
	// they're used to configure the RouteMap
	ExchangeFactory func(*RouteMap, http.ResponseWriter, *http.Request, httprouter.Params) Exchanger

	routeEntry struct {
		Name, Path string
		Resource
	}

	// RouteEntryBuilder is used to provide a concise way to define a route entry
	// to BuildRouteMap (c.f.)
	RouteEntryBuilder func(name string, path string, resource Resource)

	// RouteMap is a list of entries for routing
	RouteMap []routeEntry

	// A Resource bundles up the exchangers that deal with a kind of resources
	// (n.b. that properly, URL == resource, so a URL pattern == many resources
	Resource interface{}

	// Getable tags ResourceFamilies that respond to GET
	Getable interface {
		Get(*RouteMap, http.ResponseWriter, *http.Request, httprouter.Params) Exchanger
	}

	// Putable tags ResourceFamilies that respond to PUT
	Putable interface {
		Put(*RouteMap, http.ResponseWriter, *http.Request, httprouter.Params) Exchanger
	}

	// Deleteable tags ResourceFamilies that respond to DELETE
	Deleteable interface {
		Delete(*RouteMap, http.ResponseWriter, *http.Request, httprouter.Params) Exchanger
	}

	// Optionsable tags ResourceFamilies that respond to OPTIONS
	Optionsable interface {
		Options(*RouteMap, http.ResponseWriter, *http.Request, httprouter.Params) Exchanger
	}
	/*
		Postable interface {
			Post() Exchanger
		}
		// also consider Headable or Patchable
		// which maybe should be named "SpecializedHead" or something
		// Note that Patchable and SpecialPatch should be separate
		// The former means "there is data format that reasonably represents
		// a transform from the current GET into a reasonable PUT"
		// The latter means "thanks, but we'll handle the PATCH"
		// Likewise, maybe there should be a way for a RF to override the
		// PUT conditional behavior
	*/

	defaultOptionsExchanger struct {
		methods []string
	}
)

// BuildRouteMap receives a function that will define the route map.
// It passes in a constructor for route entries, and appends the ones constructed
// to the RouteMap.
//   BuildRouteMap(func(re RouteEntryBuilder){
//     re("me", "/author", newAuthorResource())
//   })
// This probably deserves a full example.
func BuildRouteMap(f func(RouteEntryBuilder)) *RouteMap {
	rm := RouteMap{}
	res := func(name, path string, res Resource) {
		rm = append(rm,
			routeEntry{
				Name:     name,
				Path:     path,
				Resource: res,
			})
	}
	f(res)
	return &rm
}

func (rm *RouteMap) buildMetaHandler(r *httprouter.Router, ls logging.LogSink) *MetaHandler {
	ph := &StatusMiddleware{LogSink: ls, gatelatch: os.Getenv("GATELATCH")}
	mh := &MetaHandler{
		routeMap:      rm,
		router:        r,
		statusHandler: ph,
		LogSink:       ls,
	}
	mh.InstallPanicHandler()

	return mh
}

// BuildRouter builds a returns an http.Handler based on some constant configuration
func (rm *RouteMap) BuildRouter(ls logging.LogSink) http.Handler {
	r := httprouter.New()
	mh := rm.buildMetaHandler(r, ls)

	for _, e := range *rm {
		get, canGet := e.Resource.(Getable)
		put, canPut := e.Resource.(Putable)
		del, canDel := e.Resource.(Deleteable)
		opt, canOpt := e.Resource.(Optionsable)

		if canGet {
			r.Handle("GET", e.Path, mh.GetHandling(e.Name, get.Get))
			r.Handle("HEAD", e.Path, mh.HeadHandling(e.Name, get.Get))
		}
		if canPut {
			r.Handle("PUT", e.Path, mh.PutHandling(e.Name, put.Put))
		}
		if canDel {
			r.Handle("DELETE", e.Path, mh.DeleteHandling(e.Name, del.Delete))
		}
		if canOpt {
			r.Handle("OPTIONS", e.Path, mh.OptionsHandling(e.Name, opt.Options))
		} else {
			r.Handle("OPTIONS", e.Path, mh.OptionsHandling(e.Name, defaultOptions(e.Resource)))
		}
	}

	return r
}

func defaultOptions(res Resource) ExchangeFactory {
	ex := &defaultOptionsExchanger{methods: []string{"OPTIONS"}}

	if _, can := res.(Getable); can {
		ex.methods = append(ex.methods, "GET", "HEAD")
	}
	if _, can := res.(Putable); can {
		ex.methods = append(ex.methods, "PUT")
	}
	if _, can := res.(Deleteable); can {
		ex.methods = append(ex.methods, "DELETE")
	}

	return func(*RouteMap, http.ResponseWriter, *http.Request, httprouter.Params) Exchanger {
		return ex
	}
}

func (doex *defaultOptionsExchanger) Exchange() (interface{}, int) {
	return doex.methods, 200
}

// SingleExchanger returns a single exchanger for the given exchange factory
// and injector factory. Can be useful in testing or trickier integrations.
func (rm *RouteMap) SingleExchanger(factory ExchangeFactory, gf func() Injector, ls logging.LogSink) Exchanger {
	r := httprouter.New()
	w := httptest.NewRecorder()
	rq := httptest.NewRequest("GET", "/", nil)

	mh := rm.buildMetaHandler(r, ls)

	return mh.injectedHandler(factory, w, rq, httprouter.Params{})
}

// KV (Key/Value) is a convenience type for URIFor.
type KV []string

// ByName returns the routeEntry named name and true if it exists or a zero
// routeEntry and false otherwise.
func (rm RouteMap) byName(name string) (routeEntry, bool) {
	for _, e := range rm {
		if e.Name == name {
			return e, true
		}
	}
	return routeEntry{}, false
}

// URIFor returns the URI (relative to the root) for the named route using
// pathParams and kv to fill out path parameters and query values respectively.
func (rm RouteMap) URIFor(name string, pathParams map[string]string, kvs ...KV) (string, error) {
	r, ok := rm.byName(name)
	if !ok {
		return "", fmt.Errorf("no route named %q", name)
	}
	u, err := url.ParseRequestURI(r.Path)
	if err != nil {
		return "", fmt.Errorf("error parsing route URI for %q: %s", name, err)
	}

	// Calculate query string.
	params := url.Values{}
	for _, kv := range kvs {
		params.Add(kv[0], kv[1])
	}

	query := ""
	if len(params) > 0 {
		query = "?" + url.Values(params).Encode()
	}

	// Special case for root, return early.
	if u.Path == "/" {
		return u.String() + query, nil
	}

	// For non-root calculate path based on path params.
	pathParts := strings.Split(u.Path, "/")
	pathParts = pathParts[1:]
	for i, part := range pathParts {
		if part[0] != ':' {
			continue
		}
		part = part[1:]
		value, ok := pathParams[part]
		if !ok {
			return "", fmt.Errorf("no path param for :%s", part)
		}
		pathParts[i] = url.PathEscape(value)
	}
	return "/" + strings.Join(pathParts, "/") + query, nil
}

// FQDN of the URIFor prepends the hostname.
func (rm RouteMap) FullURIFor(hostName string, name string, pathParams map[string]string, kvs ...KV) (string, error) {
	uri, err := rm.URIFor(name, pathParams, kvs...)
	return hostName + uri, err
}
