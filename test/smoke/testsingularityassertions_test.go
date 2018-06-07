//+build smoke

package smoke

import (
	"fmt"
	"testing"

	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
)

func assertActiveStatus(t *testing.T, f TestFixture, did sous.DeploymentID) {
	req := f.Singularity.GetRequestForDeployment(t, did)
	gotStatus := req.State
	wantStatus := dtos.SingularityRequestParentRequestStateACTIVE
	if gotStatus != wantStatus {
		t.Fatalf("got status %v; want %v", gotStatus, wantStatus)
	}
}

func assertSingularityRequestTypeScheduled(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	req := f.Singularity.GetRequestForDeployment(t, did)
	gotType := req.Request.RequestType
	wantType := dtos.SingularityRequestRequestTypeSCHEDULED
	if gotType != wantType {
		t.Errorf("got request type %v; want %v", gotType, wantType)
	}
}

func assertSingularityRequestTypeService(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	req := f.Singularity.GetRequestForDeployment(t, did)
	gotType := req.Request.RequestType
	wantType := dtos.SingularityRequestRequestTypeSERVICE
	if gotType != wantType {
		t.Errorf("got request type %v; want %v", gotType, wantType)
	}
}

func assertNilHealthCheckOnLatestDeploy(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	dep := f.Singularity.GetLatestDeployForDeployment(t, did)
	gotHealthcheck := dep.Deploy.Healthcheck
	if gotHealthcheck != nil {
		t.Fatalf("got Healthcheck = %v; want nil", gotHealthcheck)
	}
}

func assertUserOnLatestDeploy(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	dep := f.Singularity.GetLatestDeployForDeployment(t, did)
	if dep.DeployMarker.User != fmt.Sprintf("sous_%s", f.UserEmail) {
		t.Errorf("got user %s; want sous_%s", dep.DeployMarker.User, f.UserEmail)
	}
}

func assertNonNilHealthCheckOnLatestDeploy(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	dep := f.Singularity.GetLatestDeployForDeployment(t, did)
	gotHealthcheck := dep.Deploy.Healthcheck
	if gotHealthcheck == nil {
		t.Fatalf("got nil Healthcheck")
	}
}
