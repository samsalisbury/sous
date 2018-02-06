package singularity

import (
	"testing"
	"errors"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/opentable/sous/lib"
)

func TestDeployerMessage(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()
	pair := baseDeployablePair()
	requestID := "12345"
	taskData := &singularityTaskData{
		requestID: requestID,
	}


	reportDeployerMessage("test", pair, nil, taskData, nil, logging.InformationLevel, logger)

	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)
	assert.Equal(t, logCalls[0].PassedArgs().Get(0), logging.InformationLevel)

	logMessage := logCalls[0].PassedArgs().Get(1).(deployerMessage)

	expectedFields := map[string]interface{}{
		"@loglov3-otl": "sous-rectifier-singularity-v1",
		"request-id":   requestID,
		"diffs": "",
	}

	logging.AssertMessageFields(t, logMessage, logging.StandardDeployerFields("sous-prior","sous-post"), expectedFields)

	//weak check on WriteToConsole
	consoleCalls := control.CallsTo("Console")
	require.Len(t, consoleCalls, 1)
}

func TestDeployerMessageNilCheck(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()

	reportDeployerMessage("test", nil, nil, nil, nil, logging.InformationLevel, logger)

	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)
	assert.Equal(t, logCalls[0].PassedArgs().Get(0), logging.InformationLevel)

	logMessage := logCalls[0].PassedArgs().Get(1).(deployerMessage)

	expectedFields := map[string]interface{}{
		"@loglov3-otl": "sous-rectifier-singularity-v1",
		"diffs": "",
	}

	logging.AssertMessageFields(t, logMessage, logging.StandardVariableFields, expectedFields)
}

func TestDeployerMessageError(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()
	pair := baseDeployablePair()
	requestID := "12345"
	taskData := &singularityTaskData{
		requestID: requestID,
	}
	err := errors.New("Test error")


	reportDeployerMessage("test", pair, nil, taskData, err, logging.InformationLevel, logger)

	logCalls := control.CallsTo("LogMessage")
	logMessage := logCalls[0].PassedArgs().Get(1).(deployerMessage)

	expectedFields := map[string]interface{}{
		"@loglov3-otl": "sous-rectifier-singularity-v1",
		"request-id":   requestID,
		"error": "Test error",
		"diffs": "",
	}

	logging.AssertMessageFields(t, logMessage, logging.StandardDeployerFields("sous-prior","sous-post"), expectedFields)
}

func TestDeployerMessageDiffs(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()
	pair := baseDeployablePair()
	requestID := "12345"
	taskData := &singularityTaskData{
		requestID: requestID,
	}
	diffs := []string{"test", "test1", "test2"}


	reportDeployerMessage("test", pair, diffs, taskData, nil, logging.InformationLevel, logger)

	logCalls := control.CallsTo("LogMessage")
	logMessage := logCalls[0].PassedArgs().Get(1).(deployerMessage)

	expectedFields := map[string]interface{}{
		"@loglov3-otl": "sous-rectifier-singularity-v1",
		"request-id":   requestID,
		"diffs": "test\ntest1\ntest2",
	}

	logging.AssertMessageFields(t, logMessage, logging.StandardDeployerFields("sous-prior","sous-post"), expectedFields)
}

func TestDiffResolutionMessage(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()

	diffRes := sous.DiffResolution{
		DeploymentID: sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "repo/marker",
					Dir:  "dir/marker",
				},
				Flavor: "thai",
			},
			Cluster:    "pp-sf",
		},
		Desc:         "description goes here",
		Error:        &sous.ErrorWrapper{
			MarshallableError: sous.MarshallableError{
				Type:   "bad",
				String: "error",
			},
		},
	}


	reportDiffResolutionMessage("test", &diffRes, logging.InformationLevel, logger)

	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)
	assert.Equal(t, logCalls[0].PassedArgs().Get(0), logging.InformationLevel)

	logMessage := logCalls[0].PassedArgs().Get(1).(diffResolutionMessage)

	expectedFields := map[string]interface{}{
		"@loglov3-otl": "sous-rectifier-singularity-v1",
		"deployment-id":   diffRes.DeploymentID.String(),
		"diffresolution-desc": string(diffRes.Desc),
		"error": diffRes.Error.String,
	}

	logging.AssertMessageFields(t, logMessage, logging.StandardVariableFields, expectedFields)

	//weak check on WriteToConsole
	consoleCalls := control.CallsTo("Console")
	require.Len(t, consoleCalls, 1)
}