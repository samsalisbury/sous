package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	defaultResource struct {
		locator ComponentLocator
	}
	getDefaultHandler struct {
		routeMap restful.RouteMap
	}

	Default struct {
		Paths []string
	}
)

func newDefaultResource(loc ComponentLocator) *defaultResource {
	return &defaultResource{locator: loc}
}

func (dr *defaultResource) Get(rm *restful.RouteMap, ls logging.LogSink, rw http.ResponseWriter, r *http.Request, p httprouter.Params) restful.Exchanger {
	return &getDefaultHandler{
		routeMap: *rm,
	}
}

func (gdh *getDefaultHandler) Exchange() (interface{}, int) {
	paths := make([]string, 0)
	for _, re := range gdh.routeMap {
		paths = append(paths, re.Path)
	}
	return Default{
		Paths: paths,
	}, 200
}
