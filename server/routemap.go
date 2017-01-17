package server

import (
	"net/http"
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

	// A Resource bundles up the exchangers that deal with a kind of resources
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
	}

	return r
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
