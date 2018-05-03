package sous

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// An HTTPNameInserter sends its inserts to the configured HTTP server
type HTTPNameInserter struct {
	serverURL *url.URL
	http.Client
}

// NewHTTPNameInserter creates a new HTTPNameInserter
func NewHTTPNameInserter(server string) (*HTTPNameInserter, error) {
	u, err := url.Parse(server)
	return &HTTPNameInserter{
		serverURL: u,
	}, errors.Wrapf(err, "new artifact name inserter")
}

// Insert implements Inserter for HTTPNameInserter
func (hni *HTTPNameInserter) Insert(sid SourceID, in, etag string, qs []Quality) error {
	url, err := hni.serverURL.Parse("./artifact")
	if err != nil {
		return errors.Wrapf(err, "http insert name: %s for %v", in, sid)
	}
	url.RawQuery = sid.QueryValues().Encode()
	art := &BuildArtifact{Name: in, Type: "docker", Qualities: qs}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err = enc.Encode(art)
	if err != nil {
		return errors.Wrapf(err, "http insert name %s, encoding %v", in, art)
	}

	req, err := http.NewRequest("PUT", url.String(), buf)
	if err != nil {
		return errors.Wrapf(err, "http insert name %s, building request for %s/%v", in, url, art)
	}

	rz, err := hni.Client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "http insert name %s, sending %v", in, req)
	}
	if rz.StatusCode >= 200 && rz.StatusCode < 300 {
		return nil
	}
	return errors.Errorf("Received %s when attempting %v", rz.Status, req)
}
