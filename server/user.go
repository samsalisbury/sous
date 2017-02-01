package server

import (
	"net/http"
	"strings"

	"github.com/opentable/sous/lib"
)

// ClientUser is the sous.User configured in the calling client.
type ClientUser sous.User

// parseHeaders parses a Headers from the headers of a HTTP request.
func getUser(req *http.Request) ClientUser {
	var user ClientUser
	userString := req.Header.Get("Sous-User")
	if userString == "" {
		user.Name = "Anonymous"
	}
	parts := strings.SplitN(userString, "<", 2)
	user.Name = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		user.Email = strings.TrimSpace(strings.TrimSuffix(parts[1], ">"))
	}
	return user
}
