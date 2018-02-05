package singularity

import (
	"testing"
	"errors"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	}

	logging.AssertMessageFields(t, logMessage, logging.StandardDeployerFields("sous-prior","sous-post"), expectedFields)

	//weak check on WriteToConsole
	consoleCalls := control.CallsTo("Console")
	require.Len(t, consoleCalls, 1)
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
	}

	logging.AssertMessageFields(t, logMessage, logging.StandardDeployerFields("sous-prior","sous-post"), expectedFields)
}