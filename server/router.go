package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/graph"
)

type (
	// A Exchanger has an Exchange method - which is presumed to write to an
	// injected ResponseWriter
	Exchanger interface {
		Exchange()
	}

	ExchangeFactory func() Exchanger

	routeEntry struct {
		Name     string
		Methods  []string
		Path     string
		Exchange ExchangeFactory
	}

	// RouteMap is a list of entries for routing
	RouteMap []routeEntry
)

// RouteMap is the configuration of route for the application
const (
	DefaultMethods = []string{"GET", "POST", "PUT", "DELETE"}

	SousRouteMap = RouteMap{
		{"gdm", []string{"GET"}, "/gdm", GDMHandling},
	}
)

// BuildRouter builds a returns an http.Handler based on some constant configuration
func BuildRouter(rm RouteMap, gr graph.SousGraph) http.Handler {
	r := httprouter.New()

	for _, e := range rm {
		meths := e.Methods
		if len(meths) == 0 {
			meths = DefaultMethods
		}
		for _, m := range meths {
			r.Handle(m, e.Path, Handling(gr, e.Exchange))
		}
	}

	return r
}

// PathFor constructs a URL which should route back to the named route, with
// supplied parameters
func PathFor(name string, params map[string]string) (string, error) {
	for _, e := range RouteMap {
		if e.Name == name {
			// TODO actually handle params
			return e.Path, nil
		}
	}
	return "", errors.Newf("No route found for name %q", name)
}

// Handling (sometimes) updates the local copy of the GDM and formats it
func Handling(main *graph.SousGraph, factory ExchangeFactory) httprouter.Handle {
	func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		g := main.Clone()
		g.Add(w)
		g.Add(r)
		g.Add(p)

		h := factory()
		g.Inject(h)
		h.Go()
	}
}
