package server

import (
	"testing"

	"github.com/nyarly/spies"
	sous "github.com/opentable/sous/lib"
)

func TestGetStateDeployments(t *testing.T) {
	cm, ctrl := sous.NewClusterManagerSpy()
	ex := &GETStateDeployments{
		cluster:     cm,
		clusterName: "test-cluster",
	}
	deps := sous.NewDeployments(
		sous.DeploymentFixture("sequenced-repo"),
		sous.DeploymentFixture("sequenced-repo"),
		sous.DeploymentFixture("sequenced-repo"),
	)
	ctrl.MatchMethod("ReadCluster", spies.AnyArgs, deps, nil)

	data, status := ex.Exchange()

	if status != 200 {
		t.Fatalf("Expected 200 status, got %d", status)
	}

	gdm, is := data.(GDMWrapper)
	if !is {
		t.Fatalf("Expected response body to be a GDMWrapper, was %T", data)
	}

	if len(ctrl.CallsTo("ReadCluster")) == 0 {
		t.Errorf("No calls to ReadCluster")
	}

	if len(gdm.Deployments) != 3 {
		t.Errorf("Expected 3 Deployments, got %d", len(gdm.Deployments))
	}

}

func TestPutStateDeployments(t *testing.T) {
}

func TestRoundtripStateDeployments(t *testing.T) {
}
