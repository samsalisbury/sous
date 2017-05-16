package singularity

import (
	"testing"

	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
)

func TestStripMetadata(t *testing.T) {
	logTempl := "Got:%s Expected:%s"
	tbl := make(map[string]string)
	tbl["1.2.3-rc5"] = "1.2.3-rc5"
	tbl["1.0.1"] = "1.0.1"
	tbl["7.8.3-prerelease+METADATA"] = "7.8.3-prerelease"
	for in, out := range tbl {
		t.Logf("Stripping Metadata: %s", in)
		s := stripMetadata(in)
		if s != out {
			t.Fatalf(logTempl, s, out)
		} else {
			t.Logf(logTempl, s, out)
		}
	}
}

func TestSanitizeDeployID(t *testing.T) {
	logTempl := "Got:%s Expected:%s"
	tbl := make(map[string]string)
	tbl["this-has-dashes.and.dots"] = "this_has_dashes_and_dots"
	tbl["forward/slashes"] = "forward_slashes"
	tbl["proper_underscore"] = "proper_underscore"

	for in, out := range tbl {
		t.Logf("Sanitizing: %s", in)
		s := SanitizeDeployID(in)
		if s != out {
			t.Fatalf(logTempl, s, out)
		} else {
			t.Logf(logTempl, s, out)
		}
	}
}

func TestStripDeployID(t *testing.T) {
	logTempl := "Got:%s Expected:%s"
	tbl := make(map[string]string)
	tbl["this-has-dashes.and.dots"] = "thishasdashesanddots"
	tbl["forward/slashes"] = "forwardslashes"
	tbl["proper_underscore"] = "proper_underscore"

	for in, out := range tbl {
		t.Logf("Stripping Illegal Characters: %s", in)
		s := StripDeployID(in)
		if s != out {
			t.Fatalf(logTempl, s, out)
		} else {
			t.Logf(logTempl, s, out)
		}
	}
}

func TestDetermineRequestType(t *testing.T) {
	pairs := []struct {
		sous.ManifestKind
		dtos.SingularityRequestRequestType
	}{
		{sous.ManifestKindService, dtos.SingularityRequestRequestTypeSERVICE},
		{sous.ManifestKindWorker, dtos.SingularityRequestRequestTypeWORKER},
		{sous.ManifestKindOnDemand, dtos.SingularityRequestRequestTypeON_DEMAND},
		{sous.ManifestKindScheduled, dtos.SingularityRequestRequestTypeSCHEDULED},
		{sous.ManifestKindOnce, dtos.SingularityRequestRequestTypeRUN_ONCE},
	}

	for _, pair := range pairs {
		srrt, err := determineRequestType(pair.ManifestKind)
		if err != nil {
			t.Errorf("Error from determineRequestType: %v", err)
			continue
		}
		if srrt != pair.SingularityRequestRequestType {
			t.Errorf("Got %v expected %v.", srrt, pair.SingularityRequestRequestType)
		}

		// Should test determineManifestKind, but it's more annoying

	}
}

func TestFailOnNilBuildArtifact(t *testing.T) {
	r := sous.NewDummyRegistry()
	d := sous.Deployable{}
	ra := NewRectiAgent(r)
	err := ra.Deploy(d, "testReq")
	if err != nil {
		t.Logf("Correctly returned an error upon encountering: %#v", err)
	} else {
		t.Fatal("Deploy did not return an error when given a sous.Deployable with an empty BuildArtifact")
	}
}

func TestContainerStartupOptions(t *testing.T) {
	checkReadyPath := "/use-this-route"
	checkReadyTimeout := 45

	mockStatus := sous.DeployStatus(sous.DeployStatusPending)
	d := sous.Deployable{
		mockStatus,
		&sous.Deployment{},
		&sous.BuildArtifact{},
	}

	d.ClusterName = "TestContainerStartupOptionsCluster"
	d.Startup.CheckReadyURIPath = &checkReadyPath
	d.Startup.Timeout = &checkReadyTimeout

	dr, err := buildDeployRequest(d, "fake-request-id", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	tmpl := "expected:%s got:%s"
	if dr.Deploy.HealthcheckUri != checkReadyPath {
		t.Fatalf(tmpl, checkReadyPath, dr.Deploy.HealthcheckUri)
	} else {
		t.Logf(tmpl, checkReadyPath, dr.Deploy.HealthcheckUri)
	}

	tmpl = "expected:%d got:%d"
	if dr.Deploy.DeployHealthTimeoutSeconds != int64(checkReadyTimeout) {
		t.Fatalf(tmpl, checkReadyTimeout, dr.Deploy.DeployHealthTimeoutSeconds)
	} else {
		t.Logf(tmpl, checkReadyTimeout, dr.Deploy.DeployHealthTimeoutSeconds)
	}

}
