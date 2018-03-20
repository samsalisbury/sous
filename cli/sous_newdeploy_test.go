package cli

import (
	"fmt"
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

/*
* 	Updater interface {
		Update(body Comparable, headers map[string]string) (UpdateDeleter, error)
	}

	// A Deleter captures the state of a retrieved resource so that it can be later deleted.
	Deleter interface {
		Delete(headers map[string]string) error
	}

	// An UpdateDeleter allows for a given resource to be updated or deleted.
	UpdateDeleter interface {
		Updater
		Deleter
		Location() string
	}
*/

var log, _ = logging.NewLogSinkSpy()
var client, _ = restful.NewClient("", log, nil)
var location = "127.0.0.1:8888/deploy-queue-item?action=bb836990-5ab2-4eab-9f52-ad3fd555539b&cluster=dev-ci-sf&flavor=&offset=&repo=github.com%2Fopentable%2Fsous-demo"

type MyMockedHTTPClient struct {
	mock.Mock
	body dto.R11nResponse
}

func (m *MyMockedHTTPClient) SetRZBody(body dto.R11nResponse) {
	m.body = body
}

func (m *MyMockedHTTPClient) Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, headers map[string]string) (restful.UpdateDeleter, error) {
	args := m.Called(urlPath, qParms, rzBody, headers)
	//couldn't just assign rzBody to m.body, would never take value, so had to explicitly set
	if iSingleDeployResponse, ok := rzBody.(*dto.R11nResponse); ok {
		iSingleDeployResponse.QueuePosition = m.body.QueuePosition
		iSingleDeployResponse.Resolution = &sous.DiffResolution{}
		iSingleDeployResponse.Resolution.Desc = m.body.Resolution.Desc
	}
	//rzBody = m.body
	//rzBody.(SingleDeployResponse).QueuePosition = -1
	//rzBody.(SingleDeployResponse).DiffResolution.Desc = m.body.DiffResolution.Desc
	return args.Get(0).(restful.UpdateDeleter), args.Error(1)
}

func (m *MyMockedHTTPClient) Create(urlPath string, qParms map[string]string, rqBody interface{}, headers map[string]string) (restful.UpdateDeleter, error) {
	args := m.Called(urlPath, qParms, rqBody, headers)
	return args.Get(0).(restful.UpdateDeleter), args.Error(1)
}

func TestPollDeployQueue_success(t *testing.T) {

	httpClient := new(MyMockedHTTPClient)

	deployResult := dto.R11nResponse{}
	deployResult.QueuePosition = -1
	diffResolution := &sous.DiffResolution{}
	diffResolution.Desc = sous.ResolutionType("created")
	deployResult.Resolution = diffResolution

	httpClient.SetRZBody(deployResult)

	updateDeleter := new(MyMockedUpdateDeleter)

	httpClient.On("Retrieve", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(updateDeleter, nil)

	location = "http://" + location

	result := PollDeployQueue(location, httpClient, 1, log)
	assert.Equal(t, 0, result.ExitCode(), "This should succeed")

}

func Test_DeployQueueItem(t *testing.T) {
	response := dto.R11nResponse{}
	location = "http://" + location

	updater, err := client.Retrieve(location, nil, &response, nil)
	fmt.Println("err : ", err)
	fmt.Println("updater : ", updater)
	fmt.Println("response : ", response)

}

func TestPollDeployQueue_fail(t *testing.T) {
	location = "http://" + location

	client, _ := restful.NewClient(location, log, nil)
	result := PollDeployQueue(location, client, 1, log)

	t.Logf("poll result %v", result)
	assert.Equal(t, 70, result.ExitCode(), "This should fail")
}

func Test_PollDeployQueueBrokenURL(t *testing.T) {
	brokenLocation := "http://never-going2.wrk/test"

	result := PollDeployQueue(brokenLocation, client, 10, log)
	fmt.Printf("result : %v", result)
	assert.Equal(t, 70, result.ExitCode(), "should map to internal error")
}
