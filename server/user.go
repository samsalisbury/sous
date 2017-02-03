package server

import (
	"net/http"

	"github.com/opentable/sous/lib"
)

// ClientUser is the sous.User configured in the calling client.
type ClientUser sous.User

// getUser parses a ClientUser from the headers of a HTTP request.
func getUser(req *http.Request) ClientUser {
	// Maybe we want to check this user isn't empty, eventually.
	return ClientUser{
		Name:  req.Header.Get("Sous-User-Name"),
		Email: req.Header.Get("Sous-User-Email"),
	}
}
