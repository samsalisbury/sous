//+build smoke

package smoke

import (
	"fmt"
	"testing"
	"time"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/swaggering"
)

func assertActiveStatus(t *testing.T, f *TestFixture, reqID string) {
	req := f.Singularity.MustGetRequestForDeployment(t, reqID)
	gotStatus := req.State
	wantStatus := dtos.SingularityRequestParentRequestStateACTIVE
	if gotStatus != wantStatus {
		t.Fatalf("got status %v; want %v", gotStatus, wantStatus)
	}
}

func assertSingularityRequestID(t *testing.T, f *TestFixture, reqID string, want string) {
	t.Helper()
	req := f.Singularity.MustGetRequestForDeployment(t, reqID)
	got := req.Request.Id
	if got != want {
		t.Errorf("got request ID %q; want %q", got, want)
	}
}

func assertSingularityRequestTypeScheduled(t *testing.T, f *TestFixture, reqID string) {
	t.Helper()
	req := f.Singularity.MustGetRequestForDeployment(t, reqID)
	gotType := req.Request.RequestType
	wantType := dtos.SingularityRequestRequestTypeSCHEDULED
	if gotType != wantType {
		t.Errorf("got request type %v; want %v", gotType, wantType)
	}
}

func assertSingularityRequestTypeService(t *testing.T, f *TestFixture, reqID string) {
	t.Helper()
	req := f.Singularity.MustGetRequestForDeployment(t, reqID)
	gotType := req.Request.RequestType
	wantType := dtos.SingularityRequestRequestTypeSERVICE
	if gotType != wantType {
		t.Errorf("got request type %v; want %v", gotType, wantType)
	}
}

func assertNilHealthCheckOnLatestDeploy(t *testing.T, f *TestFixture, reqID string) {
	t.Helper()
	dep := f.Singularity.MustGetLatestDeployForDeployment(t, reqID)
	gotHealthcheck := dep.Deploy.Healthcheck
	if gotHealthcheck != nil {
		t.Fatalf("got Healthcheck = %v; want nil", gotHealthcheck)
	}
}

func assertUserOnLatestDeploy(t *testing.T, f *TestFixture, reqID string) {
	t.Helper()
	dep := f.Singularity.MustGetLatestDeployForDeployment(t, reqID)
	if dep.DeployMarker.User != fmt.Sprintf("sous_%s", f.UserEmail) {
		t.Errorf("got user %s; want sous_%s", dep.DeployMarker.User, f.UserEmail)
	}
}

func assertNonNilHealthCheckOnLatestDeploy(t *testing.T, f *TestFixture, reqID string) {
	t.Helper()
	dep := f.Singularity.MustGetLatestDeployForDeployment(t, reqID)
	gotHealthcheck := dep.Deploy.Healthcheck
	if gotHealthcheck == nil {
		t.Fatalf("got nil Healthcheck")
	}
}

func assertRequestDoesNotExist(t *testing.T, f *TestFixture, reqID string) {
	t.Helper()
	var err error
	waitFor(t, "request to be deleted", time.Minute, time.Second, func() error {
		_, err = f.Singularity.GetRequestForDeployment(t, reqID)
		if err == nil {
			return fmt.Errorf("request %q still exists", reqID)
		}
		return nil
	})
	reqErr, ok := err.(*swaggering.ReqError)
	if !ok {
		t.Fatalf("unable to assert if request exists or not: %s", err)
	}
	if reqErr.Status != 404 {
		t.Fatalf("unexpected status code %d (want 404)", reqErr.Status)
	}
}
