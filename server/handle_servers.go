package server

import "github.com/opentable/sous/config"

type (
	ServerListResource struct{}

	ServerListHandler struct {
		Config *config.Config
	}

	serverListData struct {
		Servers []string
	}
)

// Get implements Getable on ServerListResource
func (slr *ServerListResource) Get() Exchanger { return &ServerListHandler{} }

// Exchange implements Exchanger on ServerListHandler
func (slh *ServerListHandler) Exchange() (interface{}, int) {
	data := serverListData{Servers: make([]string, len(slh.Config.SiblingURLs))}
	copy(data.Servers, slh.Config.SiblingURLs)
	return data, 200
}
