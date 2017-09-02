package server

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// ServerListResource dispatches /servers
	ServerListResource struct {
		context ComponentLocator
	}

	// ServerListHandler handles GET for /servers
	ServerListHandler struct {
		Config *config.Config
	}

	// ServerListUpdater handles PUT for /servers
	ServerListUpdater struct {
		*http.Request
		Config *config.Config
		Log    logging.LogSet
	}
)

func newServerListResource(context ComponentLocator) *ServerListResource {
	return &ServerListResource{context: context}
}

// Get implements Getable on ServerListResource, which marks it as accepting GET requests
func (slr *ServerListResource) Get(http.ResponseWriter, *http.Request, httprouter.Params) restful.Exchanger {
	return &ServerListHandler{
		Config: slr.context.Config,
	}
}

// Put implements Putable on ServerListResource, which marks is as accepting PUT requests
func (slr *ServerListResource) Put(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &ServerListUpdater{
		Config:  slr.context.Config,
		Log:     slr.context.LogSet,
		Request: req,
	}
}

// Exchange implements restful.Exchanger on ServerListHandler
func (slh *ServerListHandler) Exchange() (interface{}, int) {
	data := ServerListData{Servers: []NameData{}}
	for name, url := range slh.Config.SiblingURLs {
		data.Servers = append(data.Servers, NameData{ClusterName: name, URL: url})
	}
	return data, 200
}

// Exchange implements restful.Exchanger on ServerListUpdater
func (slh *ServerListUpdater) Exchange() (interface{}, int) {
	dec := json.NewDecoder(slh.Request.Body)
	data := ServerListData{Servers: []NameData{}}
	dec.Decode(&data)

	slh.Log.Vomit.Printf("Updating server list to: %#v", data)

	if slh.Config.SiblingURLs == nil {
		slh.Config.SiblingURLs = make(map[string]string)
	}

	for _, server := range data.Servers {
		slh.Config.SiblingURLs[server.ClusterName] = server.URL
	}

	return data, 200
}
