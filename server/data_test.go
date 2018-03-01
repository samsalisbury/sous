package server

import (
	"net/http"
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestGDMWrapperAddHeaders(t *testing.T) {
	wrapper := GDMWrapper{
		Deployments: []*sous.Deployment{
			sous.DeploymentFixture("sequenced-repo"),
			sous.DeploymentFixture("sequenced-repo"),
			sous.DeploymentFixture("sequenced-repo"),
		},
	}

	headers := http.Header{}

	wrapper.AddHeaders(headers)

	if !strings.HasPrefix(headers.Get("Etag"), "w/") {
		t.Errorf("Expected Etag with prefix %q, got Etag %q", "w/", headers.Get("Etag"))
	}
}
