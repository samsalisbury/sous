package server

import (
	"net/url"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type pm map[string]string

func TestSousRoutes(t *testing.T) {
	test := func(er, name string, params map[string]string, kvs ...restful.KV) {
		t.Helper()
		ar, err := routemap(ComponentLocator{}).URIFor(name, params, kvs...)
		if err != nil {
			t.Fatalf("Error getting a path: %#v", err)
		}
		if ar != er {
			t.Errorf("Route bad: expected %q got %q", er, ar)
		}

	}
	test(
		"/manifest?flavor=sweet&offset=alt&repo=github.com%2Fopentable%2Fsous",
		"manifest",
		nil,
		restful.KV{"repo", "github.com/opentable/sous"},
		restful.KV{"offset", "alt"},
		restful.KV{"flavor", "sweet"},
	)
	test(
		"/manifest?offset=alt&repo=github.com%2Fopentable%2Fsous",
		"manifest",
		nil,
		restful.KV{"repo", "github.com/opentable/sous"},
		restful.KV{"offset", "alt"},
	)
	test(
		"/status",
		"status",
		nil,
	)

	// This test checks that the deploymentID.String() implementation that
	// exists at time of writing is read OK. This test will break if that
	// implementation changes.
	test(
		"/deploy-queue?DeploymentID=my-cluster%3Agithub.com%2Fopentable%2Fblah%2Csome%2Fdir~orange",
		"deploy-queue",
		nil,
		restful.KV{"DeploymentID", "my-cluster:github.com/opentable/blah,some/dir~orange"},
	)

	// This test checks that the current implementation of DeploymentID.String()
	// is read OK, and will likely be more resilient than the previous test.
	did := sous.DeploymentID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "github.com/opentable/blah",
				Dir:  "some/dir",
			},
			Flavor: "orange",
		},
		Cluster: "my-cluster",
	}
	didStr := did.String()
	t.Logf("DeploymentID string: %q", didStr)
	escapedDidStr := url.QueryEscape(didStr)
	t.Logf("DeploymentID escaped: %q", escapedDidStr)
	test(
		"/deploy-queue?DeploymentID="+escapedDidStr,
		"deploy-queue",
		nil,
		restful.KV{"DeploymentID", did.String()},
	)

	test("/health", "health", nil)

	test(
		"/single-deployment?cluster=cluster1&offset=alt&repo=github.com%2Fopentable%2Fsous",
		"single-deployment",
		nil,
		restful.KV{"repo", "github.com/opentable/sous"},
		restful.KV{"offset", "alt"},
		restful.KV{"cluster", "cluster1"},
	)
}
