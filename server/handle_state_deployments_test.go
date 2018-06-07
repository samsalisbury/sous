package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/dto"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

func TestGetStateDeployments(t *testing.T) {
	cm, ctrl := sous.NewClusterManagerSpy()
	ls, _ := logging.NewLogSinkSpy()

	ex := &GETStateDeployments{
		cluster:     cm,
		clusterName: "test-cluster",
		log:         ls,
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

	gdm, is := data.(dto.GDMWrapper)
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
	cm, ctrl := sous.NewClusterManagerSpy()
	gdm := dto.GDMWrapper{
		Deployments: []*sous.Deployment{
			sous.DeploymentFixture("sequenced-repo"),
			sous.DeploymentFixture("sequenced-repo"),
			sous.DeploymentFixture("sequenced-repo"),
		},
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(gdm)

	req, err := http.NewRequest("PUT", "", buf)
	if err != nil {
		t.Fatal("error building request", err)
	}

	ex := &PUTStateDeployments{
		cluster:     cm,
		clusterName: "test-cluster",
		req:         req,
	}

	data, status := ex.Exchange()

	wantStatus := 202
	if status != wantStatus {
		t.Fatalf("Expected %d status, got %d", wantStatus, status)
	}

	if data != nil {
		t.Fatalf("Expect nil data was %#v", data)
	}

	if len(ctrl.CallsTo("WriteCluster")) == 0 {
		t.Errorf("No calls to WriteCluster")
	}
}
