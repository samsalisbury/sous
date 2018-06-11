package singularity

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	singularity "github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/swaggering"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	ls, _ := logging.NewLogSinkSpy()
	ra := NewRectiAgent(r, ls)
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

	ls, _ := logging.NewLogSinkSpy()

	dr, err := buildDeployRequest(d, "fake-request-id", "fake-deploy-id", map[string]string{}, ls)
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

func TestDeploy_MockedSingularity(t *testing.T) {

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
	d.Deployment.Cluster = &sous.Cluster{
		BaseURL: "http://testcluster.com",
	}
	d.Deployment.User.Email = "testuser@example.com"

	ls, _ := logging.NewLogSinkSpy()

	reqID := "fake-request-id"
	depID := "fake-deploy-id"

	r := sous.NewDummyRegistry()
	ra := NewRectiAgent(r, ls)

	dummyClient, ctrl := singularity.NewDummyClient(d.Deployment.Cluster.BaseURL)
	ra.singClients[d.Deployment.Cluster.BaseURL] = dummyClient

	response := new(dtos.SingularityRequestParent)

	ctrl.FeedDTO(response, nil)
	err := ra.Deploy(d, reqID, depID)

	if err != nil {
		t.Errorf("Should complete without failure")
	}
}

func TestDeploy_User(t *testing.T) {

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
	d.Deployment.Cluster = &sous.Cluster{
		BaseURL: "http://testcluster.com",
	}
	d.Deployment.User.Email = "testuser@example.com"

	ls, _ := logging.NewLogSinkSpy()

	reqID := "fake-request-id"
	depID := "fake-deploy-id"

	r := sous.NewDummyRegistry()
	ra := NewRectiAgent(r, ls)

	myClient := new(MySingularityClient)

	user := "sous"
	if d.Deployment != nil && len(d.Deployment.User.Email) > 1 {
		user = "sous_" + d.Deployment.User.Email
	}
	qMap := make(swaggering.UrlParams)
	qMap["user"] = user

	myClient.On("DTORequest", qMap).Return(nil)

	ra.singClients[d.Deployment.Cluster.BaseURL] = myClient

	err := ra.Deploy(d, reqID, depID)

	if err != nil {
		t.Errorf("Should complete without failure")
	}

	myClient.AssertExpectations(t)
}

type MySingularityClient struct {
	mock.Mock
}

func (m *MySingularityClient) DTORequest(resourceName string, dto swaggering.DTO, method string, path string, pathParams swaggering.UrlParams, queryParams swaggering.UrlParams, body ...swaggering.DTO) error {
	args := m.Called(queryParams)
	return args.Error(0)
}

func (m *MySingularityClient) Request(resourceName string, method string, path string, pathParams swaggering.UrlParams, queryParams swaggering.UrlParams, body ...swaggering.DTO) (io.ReadCloser, error) {
	args := m.Called(queryParams)
	return ioutil.NopCloser(bytes.NewBufferString("")), args.Error(1)
}
