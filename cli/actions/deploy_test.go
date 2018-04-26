package actions

import (
	"testing"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/dto"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful/restfultest"
	"github.com/stretchr/testify/assert"
)

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

func TestPollDeployQueue_success_created(t *testing.T) {
	log, _ := logging.NewLogSinkSpy()
	httpClient, ctrl := restfultest.NewHTTPClientSpy()

	createDeployResult(ctrl, -1, "created", 2)

	sd := &Deploy{
		HTTPClient: httpClient,
		LogSink:    log,
	}

	assert.NoError(t, sd.pollDeployQueue("127.0.0.1:1234/deploy-queue-item", 1, nil))
}

func TestPollDeployQueue_success_updated(t *testing.T) {
	log, _ := logging.NewLogSinkSpy()
	httpClient, ctrl := restfultest.NewHTTPClientSpy()
	createDeployResult(ctrl, -1, "updated", 2)
	sd := &Deploy{
		HTTPClient: httpClient,
		LogSink:    log,
	}

	assert.NoError(t, sd.pollDeployQueue("127.0.0.1:1234/deploy-queue-item", 1, nil))
}

func TestPollDeployQueue_fail(t *testing.T) {
	log, _ := logging.NewLogSinkSpy()
	location := "127.0.0.1:8888/deploy-queue-item?action=bb836990-5ab2-4eab-9f52-ad3fd555539b&cluster=dev-ci-sf&flavor=&offset=&repo=github.com%2Fopentable%2Fsous-demo"

	httpClient, _ := restfultest.NewHTTPClientSpy()

	sd := &Deploy{
		HTTPClient: httpClient,
		LogSink:    log,
	}

	assert.Error(t, sd.pollDeployQueue(location, 10, nil))
}

/*
func (m *MyMockedHTTPClient) SetRZBody(body dto.R11nResponse) {
	m.body = body
}
*/

func createDeployResult(ctrl *spies.Spy, queuePosition int, resolutionType sous.ResolutionType, status sous.DeployStatus) {
	deployResult := dto.R11nResponse{
		QueuePosition: queuePosition,
		Resolution: &sous.DiffResolution{
			Desc: resolutionType,
			DeployState: &sous.DeployState{
				Status: status,
			},
		},
	}

	ud, _ := restfultest.NewUpdateSpy()
	ctrl.MatchMethod("Retrieve", spies.AnyArgs, deployResult, ud, nil)
}
