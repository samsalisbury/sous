package cli

import (
	"testing"

	"github.com/opentable/sous/dto"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MyMockedUpdateDeleter struct {
	mock.Mock
}

func (m *MyMockedUpdateDeleter) Update(body restful.Comparable, headers map[string]string) (restful.UpdateDeleter, error) {
	args := m.Called(body, headers)
	return args.Get(0).(restful.UpdateDeleter), args.Error(1)
}

func (m *MyMockedUpdateDeleter) Delete(headers map[string]string) error {
	args := m.Called(headers)
	return args.Error(0)
}

func (m *MyMockedUpdateDeleter) Location() string {
	args := m.Called()
	return args.String(0)
}

type MyMockedHTTPClient struct {
	mock.Mock
	body dto.R11nResponse
}

func (m *MyMockedHTTPClient) SetRZBody(body dto.R11nResponse) {
	m.body = body
}

var log, _ = logging.NewLogSinkSpy()

func (m *MyMockedHTTPClient) Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, headers map[string]string) (restful.UpdateDeleter, error) {
	args := m.Called(urlPath, qParms, rzBody, headers)
	//couldn't just assign rzBody to m.body, would never take value, so had to explicitly set
	if iSingleDeployResponse, ok := rzBody.(*dto.R11nResponse); ok {
		iSingleDeployResponse.QueuePosition = m.body.QueuePosition
		iSingleDeployResponse.Resolution = &sous.DiffResolution{}
		iSingleDeployResponse.Resolution.Desc = m.body.Resolution.Desc

		iSingleDeployResponse.Resolution.DeployState = &sous.DeployState{}
		iSingleDeployResponse.Resolution.DeployState.Status = m.body.Resolution.DeployState.Status

	}
	return args.Get(0).(restful.UpdateDeleter), args.Error(1)
}

func (m *MyMockedHTTPClient) Create(urlPath string, qParms map[string]string, rqBody interface{}, headers map[string]string) (restful.UpdateDeleter, error) {
	args := m.Called(urlPath, qParms, rqBody, headers)
	return args.Get(0).(restful.UpdateDeleter), args.Error(1)
}

func TestPollDeployQueue_success_created(t *testing.T) {
	httpClient := createMockHTTPClient()
	httpClient.SetRZBody(createDeployResult(-1, "created", 2))

	result := PollDeployQueue("127.0.0.1:1234/deploy-queue-item", httpClient, 1, log)
	assert.Equal(t, 0, result.ExitCode(), "created should return 0 exit code")
}

func TestPollDeployQueue_success_updated(t *testing.T) {
	httpClient := createMockHTTPClient()
	httpClient.SetRZBody(createDeployResult(-1, "updated", 2))

	result := PollDeployQueue("127.0.0.1:1234/deploy-queue-item", httpClient, 1, log)
	assert.Equal(t, 0, result.ExitCode(), "updated should return 0 exit code")
}

func createMockHTTPClient() *MyMockedHTTPClient {
	httpClient := new(MyMockedHTTPClient)
	updateDeleter := new(MyMockedUpdateDeleter)
	httpClient.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(updateDeleter, nil)

	return httpClient
}

func createDeployResult(queuePosition int, resolutionType string, status int) dto.R11nResponse {
	deployResult := dto.R11nResponse{}
	deployResult.QueuePosition = queuePosition
	diffResolution := &sous.DiffResolution{}
	diffResolution.Desc = sous.ResolutionType(resolutionType)
	deployResult.Resolution = diffResolution
	deployState := sous.DeployState{}
	deployState.Status = sous.DeployStatus(status)
	deployResult.Resolution.DeployState = &deployState
	return deployResult
}

func TestPollDeployQueue_fail(t *testing.T) {
	location := "127.0.0.1:8888/deploy-queue-item?action=bb836990-5ab2-4eab-9f52-ad3fd555539b&cluster=dev-ci-sf&flavor=&offset=&repo=github.com%2Fopentable%2Fsous-demo"

	client, _ := restful.NewClient("", log, nil)
	result := PollDeployQueue(location, client, 10, log)

	t.Logf("poll result %v", result)
	assert.Equal(t, 70, result.ExitCode(), "This should fail")
}

func TestCheckFinished(t *testing.T) {
	resolution := sous.DiffResolution{}
	resolution.Desc = sous.CreateDiff
	assert.True(t, checkFinished(resolution), "resolution should be true")

	resolution.Desc = sous.DeleteDiff
	assert.True(t, !checkFinished(resolution), "resolution should be false")

	resolution.Desc = sous.ModifyDiff
	assert.True(t, checkFinished(resolution), "resolution should be true")

	assert.True(t, !checkFinished(sous.DiffResolution{}), "empty resolution should be false")
}
