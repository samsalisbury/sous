package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
)

type (
	healthResource struct {
		locator ComponentLocator
	}

	getHealthHandler struct {
		version semv.Version
	}

	// Health is the DTO for representing the health of the Sous server
	Health struct {
		Version  string
		Revision string
	}
)

func newHealthResource(loc ComponentLocator) *healthResource {
	return &healthResource{locator: loc}
}

func (hr *healthResource) Get(*restful.RouteMap, logging.LogSink, http.ResponseWriter, *http.Request, httprouter.Params) restful.Exchanger {
	return &getHealthHandler{
		version: hr.locator.Version,
	}
}

func (ghh *getHealthHandler) Exchange() (interface{}, int) {
	return Health{
		Version:  ghh.version.Format(semv.MMPPre),
		Revision: ghh.version.Format(semv.Meta),
	}, 200
}
