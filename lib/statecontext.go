package sous

import (
	"net/http"

	"github.com/samsalisbury/semv"
)

// StateContext contains additional data about what is being read or written
// by a StateManager.
type StateContext struct {
	// User is the user a read or write is attributed to.
	User User
	// TargetManifestID is the manifest this write is expected to affect.
	// Implementations of StateWriter.WriteState should check that the
	// change being written corresponds with this manifest ID.
	TargetManifestID ManifestID
	// ClientVersion is the version of the sous client reading or writing state.
	ClientVersion semv.Version
	// Command is the command entered by a user.
	Command string
}

const TargetManifestIDHeaderKey = "Sous-Target-Manifest-ID"
const UserNameHeaderKey = "Sous-User-Name"
const UserEmailHeaderKey = "Sous-User-Email"
const ClientVersionHeaderKey = "Sous-Client-Version"
const CommandHeaderKey = "Sous-Client-Command"

// WriteHeaders writes this StateContext to a set of HTTP headers.
func (sc *StateContext) WriteHeaders(header http.Header) {
	header.Add("Sous-User-Name", sc.User.Name)
	header.Add("Sous-User-Email", sc.User.Email)
	header.Add("Sous-Target-Manifest-ID", sc.TargetManifestID.String())
}

// NewStateContextFromHTTPHeader parses a StateContext from an http.Header.
func NewStateContextFromHTTPHeader(req *http.Request) (StateContext, error) {

	clientVersionString := req.Header.Get(ClientVersionHeaderKey)
	clientVersion, err := semv.Parse(clientVersionString)
	if err != nil {
		Log.Warn.Printf("Unable to parse client version %q from header %s: %s",
			clientVersionString, ClientVersionHeaderKey, err)
	}

	midString := req.Header.Get(TargetManifestIDHeaderKey)
	mid, err := ParseManifestID(midString)
	if err != nil {
		Log.Warn.Printf("Unable to parse manifest ID %q from header %s: %s",
			midString, TargetManifestIDHeaderKey, err)
	}

	command := req.Header.Get(CommandHeaderKey)

	// Maybe we want to check this user isn't empty, eventually.
	return StateContext{
		User: User{
			Name:  req.Header.Get(UserNameHeaderKey),
			Email: req.Header.Get(UserEmailHeaderKey),
		},
		TargetManifestID: mid,
		ClientVersion:    clientVersion,
		Command:          command,
	}, nil
}
