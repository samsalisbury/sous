package server

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestPUTSingleDeploymentHandler_Exchange(t *testing.T) {

	psd := PUTSingleDeploymentHandler{
		DeploymentID:      sous.DeploymentID{},
		DeploymentIDError: nil,
		Manifest:          sous.Manifest{},
		ManifestError:     nil,
	}

	body, gotStatus := psd.Exchange()

	got, ok := body.(singleDeploymentResponse)
	if !ok {
		t.Fatalf("got a %T; want a %T", body, got)
	}

	wantStatus := 404
	if gotStatus != wantStatus {
		t.Errorf("got status %d; want %d", gotStatus, wantStatus)
	}

}
