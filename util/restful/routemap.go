package restful

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

type (
	// A Exchanger has an Exchange method - which is presumed to write to an
	// injected ResponseWriter
	Exchanger interface {
		Exchange() (interface{}, int)
	}

	// An ExchangeFactory builds an Exchanger -
	// they're used to configure the RouteMap
	ExchangeFactory func() Exchanger

	routeEntry struct {
		Name, Path string
		Resource
	}

	// RouteMap is a list of entries for routing
	RouteMap []routeEntry

	// A ResourceFamily bundles up the exchangers that deal with a kind of resources
	// (n.b. that properly, URL == resource, so a URL pattern == many resources
	Resource interface{}

	// Getable tags ResourceFamilies that respond to GET
	Getable interface {
		Get() Exchanger
	}

	// Putable tags ResourceFamilies that respond to PUT
	Putable interface {
		Put() Exchanger
	}

	// Deleteable tags ResourceFamilies that respond to DELETE
	Deleteable interface {
		Delete() Exchanger
	}

	// Optionsable tags ResourceFamilies that respond to OPTIONS
	Optionsable interface {
		Options() Exchanger
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

func (rm *RouteMap) buildMetaHandler(r *httprouter.Router, grf func() Injector) *MetaHandler {
	ph := &StatusMiddleware{}
	mh := &MetaHandler{
		graphFac:      grf,
		router:        r,
		statusHandler: ph,
	}
	mh.InstallPanicHandler()

	return mh
}

// BuildRouter builds a returns an http.Handler based on some constant configuration
func (rm *RouteMap) BuildRouter(grf func() Injector) http.Handler {
	r := httprouter.New()
	mh := rm.buildMetaHandler(r, grf)

	for _, e := range *rm {
		get, canGet := e.Resource.(Getable)
		put, canPut := e.Resource.(Putable)
		del, canDel := e.Resource.(Deleteable)
		opt, canOpt := e.Resource.(Optionsable)

		if canGet {
			r.Handle("GET", e.Path, mh.GetHandling(get.Get))
			r.Handle("HEAD", e.Path, mh.HeadHandling(get.Get))
		}
		if canPut {
			r.Handle("PUT", e.Path, mh.PutHandling(put.Put))
		}
		if canDel {
			r.Handle("DELETE", e.Path, mh.DeleteHandling(del.Delete))
		}
		if canOpt {
			r.Handle("OPTIONS", e.Path, mh.OptionsHandling(opt.Options))
		} else {
			r.Handle("OPTIONS", e.Path, mh.OptionsHandling(defaultOptions(e.Resource)))
		}
	}

	return r
}

func defaultOptions(res Resource) func() Exchanger {
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

	return func() Exchanger {
		return ex
	}
}

func (doex *defaultOptionsExchanger) Exchange() (interface{}, int) {
	return doex.methods, 200
}

// SingleExchanger returns a single exchanger for the given exchange factory
// and injector factory. Can be useful in testing or trickier integrations.
func (rm *RouteMap) SingleExchanger(factory ExchangeFactory, gf func() Injector) Exchanger {
	r := httprouter.New()
	w := httptest.NewRecorder()
	rq := httptest.NewRequest("GET", "/", nil)

	mh := rm.buildMetaHandler(r, gf)

	return mh.injectedHandler(factory, w, rq, httprouter.Params{})
}

// KV (Key/Value) is a convenience type for PathFor
type KV []string

// PathFor constructs a URL which should route back to the named route, with
// supplied parameters
func (rm *RouteMap) PathFor(name string, kvs ...KV) (string, error) {
	params := url.Values{}
	for _, kv := range kvs {
		params.Add(kv[0], kv[1])
	}

	for _, e := range *rm {
		if e.Name == name {
			// Path parameters will need some regexp magic, I think
			query := ""
			if len(params) > 0 {
				query = "?" + url.Values(params).Encode()
			}
			return e.Path + query, nil
		}
	}
	return "", errors.Errorf("No route found for name %q", name)
}
