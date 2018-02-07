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

	if strings.Index(headers.Get("Etag"), "w/") != 0 {
		t.Errorf("Expected a w/ Etag, got %q", headers.Get("Etag"))
	}
}
