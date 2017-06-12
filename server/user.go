package server

import (
	"net/http"

	"github.com/opentable/sous/lib"
)

// ClientUser is the sous.User configured in the calling client.
type ClientUser sous.StateWriteContext

// getUser parses a ClientUser from the headers of a HTTP request.
func getUser(req *http.Request) (ClientUser, error) {

	midString := req.Header.Get("Sous-Target-Manifest-ID")
	mid, err := sous.ParseManifestID(midString)
	if err != nil {
		return ClientUser{}, err
	}

	// Maybe we want to check this user isn't empty, eventually.
	return ClientUser{
		User: sous.User{
			Name:  req.Header.Get("Sous-User-Name"),
			Email: req.Header.Get("Sous-User-Email"),
		},
		TargetManifestID: mid,
	}, nil
}
