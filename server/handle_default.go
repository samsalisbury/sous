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
	}

	Default struct {
		Message string
	}
)

func newDefaultResource(loc ComponentLocator) *defaultResource {
	return &defaultResource{locator: loc}
}

func (dr *defaultResource) Get(*restful.RouteMap, logging.LogSink, http.ResponseWriter, *http.Request, httprouter.Params) restful.Exchanger {
	return &getDefaultHandler{}
}

func (gdh *getDefaultHandler) Exchange() (interface{}, int) {
	return Default{
		Message: "all good",
	}, 200
}
