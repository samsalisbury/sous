package singularity

import (
	"testing"

	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
	"github.com/stretchr/testify/assert"
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
	err := ra.Deploy(d, "testReq", "testDep")
	if err != nil {
		t.Logf("Correctly returned an error upon encountering: %#v", err)
	} else {
		t.Fatal("Deploy did not return an error when given a sous.Deployable with an empty BuildArtifact")
	}
}

func TestMapStartup(t *testing.T) {
	buildHealthcheckedDTOMap := func(t *testing.T, ready bool) dtoMap {
		depMap := dtoMap{}
		startup := sous.Startup{
			SkipCheck: ready,

			ConnectDelay:    106,
			Timeout:         107,
			ConnectInterval: 108,

			CheckReadyProtocol:        "https",
			CheckReadyURIPath:         "dummyCheckReadyURIPath",
			CheckReadyPortIndex:       114,
			CheckReadyFailureStatuses: []int{500},
			CheckReadyURITimeout:      116,
			CheckReadyInterval:        117,
			CheckReadyRetries:         118,
		}
		err := MapStartupIntoHealthcheckOptions((*map[string]interface{})(&depMap), startup)
		if err != nil {
			t.Fatalf("Received and error loading a map!")
		}
		return depMap
	}

	t.Run("Skip", func(t *testing.T) {
		dep := buildHealthcheckedDTOMap(t, true)
		assert.NotContains(t, dep, "Healthcheck")
	})

	t.Run("Don't skip", func(t *testing.T) {
		dep := buildHealthcheckedDTOMap(t, false)
		if assert.Contains(t, dep, "Healthcheck") {
			hco := dep["Healthcheck"].(*dtos.HealthcheckOptions)
			assert.Equal(t, int32(106), hco.StartupDelaySeconds)                           //ConnectDelay
			assert.Equal(t, int32(107), hco.StartupTimeoutSeconds)                         //Timeout
			assert.Equal(t, int32(108), hco.StartupIntervalSeconds)                        //ConnectInterval
			assert.Equal(t, dtos.HealthcheckOptionsHealthcheckProtocolhttps, hco.Protocol) //CheckReadyProtocol
			assert.Equal(t, "dummyCheckReadyURIPath", hco.Uri)                             //CheckReadyURIPath
			assert.Equal(t, int32(114), hco.PortIndex)                                     //CheckReadyPortIndex
			assert.Equal(t, []int32{500}, hco.FailureStatusCodes)                          //CheckReadyFailureStatuses
			assert.Equal(t, int32(116), hco.ResponseTimeoutSeconds)                        //CheckReadyURITimeout
			assert.Equal(t, int32(117), hco.IntervalSeconds)                               //CheckReadyInterval
			assert.Equal(t, int32(118), hco.MaxRetries)                                    //CheckReadyRetries
		}
	})
}

func TestContainerStartupOptions(t *testing.T) {
	checkReadyPath := "/use-this-route"
	checkReadyTimeout := 45

	mockStatus := sous.DeployStatus(sous.DeployStatusPending)
	d := sous.Deployable{
		Status:        mockStatus,
		Deployment:    &sous.Deployment{},
		BuildArtifact: &sous.BuildArtifact{},
	}

	d.ClusterName = "TestContainerStartupOptionsCluster"
	d.Startup.CheckReadyURIPath = checkReadyPath
	d.Startup.Timeout = checkReadyTimeout

	dr, err := buildDeployRequest(d, "fake-request-id", "fake-deploy-id", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	if dr.Deploy.Healthcheck.Uri != checkReadyPath {
		t.Errorf("expected:%s got:%s", checkReadyPath, dr.Deploy.Healthcheck.Uri)
	}

	if dr.Deploy.Healthcheck.StartupTimeoutSeconds != int32(checkReadyTimeout) {
		t.Errorf("expected:%d got:%d", checkReadyTimeout, dr.Deploy.Healthcheck.StartupTimeoutSeconds)
	}

}
