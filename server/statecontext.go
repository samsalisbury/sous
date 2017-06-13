package server

import (
	"net/http"

	"github.com/opentable/sous/lib"
)

// StateContext is the sous.StateContext provided by the calling client.
type StateContext sous.StateContext

// getUser parses a ClientUser from the headers of a HTTP request.
func getUser(req *http.Request) (StateContext, error) {

	midString := req.Header.Get("Sous-Target-Manifest-ID")
	mid, err := sous.ParseManifestID(midString)
	if err != nil {
		return StateContext{}, err
	}

	// Maybe we want to check this user isn't empty, eventually.
	return StateContext{
		User: sous.User{
			Name:  req.Header.Get("Sous-User-Name"),
			Email: req.Header.Get("Sous-User-Email"),
		},
		TargetManifestID: mid,
	}, nil
}
