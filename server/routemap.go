package server

import (
	"net/http"

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
	/*
		Postable interface {
			Post() Exchanger
		}
		Deleteable interface {
			Delete() Exchanger
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

// BuildRouter builds a returns an http.Handler based on some constant configuration
func (rm *RouteMap) BuildRouter(grf func() Injector) http.Handler {
	r := httprouter.New()
	ph := &StatusHandler{}
	mh := &MetaHandler{
		graphFac:      grf,
		router:        r,
		statusHandler: ph,
	}
	mh.InstallPanicHandler()

	for _, e := range *rm {
		get, canGet := e.Resource.(Getable)
		put, canPut := e.Resource.(Putable)

		if canGet {
			r.Handle("GET", e.Path, mh.GetHandling(get.Get))
			r.Handle("HEAD", e.Path, mh.HeadHandling(get.Get))
		}
		if canPut {
			r.Handle("PUT", e.Path, mh.PutHandling(put.Put))
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
