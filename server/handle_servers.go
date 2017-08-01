package server

import (
	"encoding/json"
	"net/http"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// ServerListResource dispatches /servers
	ServerListResource struct{}

	// ServerListHandler handles GET for /servers
	ServerListHandler struct {
		Config *config.Config
	}

	// ServerListUpdater handles PUT for /servers
	ServerListUpdater struct {
		*http.Request
		Config *config.Config
		Log    *logging.LogSet
	}
)

// Get implements Getable on ServerListResource
func (slr *ServerListResource) Get() restful.Exchanger { return &ServerListHandler{} }
func (slr *ServerListResource) Put() restful.Exchanger { return &ServerListUpdater{} }

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
