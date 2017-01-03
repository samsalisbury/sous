package server

import "github.com/opentable/sous/config"

type (
	// ServerListResource dispatches /servers
	ServerListResource struct{}

	// ServerListHandler handles GET for /servers
	ServerListHandler struct {
		Config *config.Config
	}

	server struct {
		URL string
	}

	serverListData struct {
		Servers []server
	}
)

// Get implements Getable on ServerListResource
func (slr *ServerListResource) Get() Exchanger { return &ServerListHandler{} }

// Exchange implements Exchanger on ServerListHandler
func (slh *ServerListHandler) Exchange() (interface{}, int) {
	data := serverListData{Servers: []server{}}
	for _, url := range slh.Config.SiblingURLs {
		data.Servers = append(data.Servers, server{URL: url})
	}
	return data, 200
}
