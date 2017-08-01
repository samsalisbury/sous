package graph

import (
	"net/http"

	"github.com/opentable/sous/server"
)

func getLiveGDM(sr StateReader) (*server.LiveGDM, error) {
	state, err := NewCurrentState(sr)
	if err != nil {
		return nil, err
	}
	gdm, err := NewCurrentGDM(state)
	if err != nil {
		return nil, err
	}
	// Ignore this error because an empty string etag is acceptable.
	etag, _ := state.GetEtag()
	return &server.LiveGDM{Etag: etag, Deployments: gdm.Deployments}, nil
}

func getUser(req *http.Request) server.ClientUser {
	// Maybe we want to check this user isn't empty, eventually.
	return server.ClientUser{
		Name:  req.Header.Get("Sous-User-Name"),
		Email: req.Header.Get("Sous-User-Email"),
	}
}
