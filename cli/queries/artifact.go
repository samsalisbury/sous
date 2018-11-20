package queries

import (
	"strings"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

// ArtifactQuery supports querying artifacts by their SourceID only.
type ArtifactQuery struct {
	Client restful.HTTPClient
	User   sous.User
}

// ByID returns the single artifact matched by sid. It returns nil, nil if there
// is no match and no error determining that.
func (q *ArtifactQuery) ByID(sid sous.SourceID) (*sous.BuildArtifact, error) {
	ba := &sous.BuildArtifact{}
	header := q.User.HTTPHeaders()

	_, err := q.Client.Retrieve("./artifact", sid.HTTPQueryMap(), ba, header)
	if err == nil {
		return ba, nil
	}
	if strings.Contains(err.Error(), "404 Not Found") {
		return nil, nil
	}
	return nil, err
}
