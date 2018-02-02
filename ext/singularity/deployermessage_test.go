package singularity

import (
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployerMessage(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()
	pair := baseDeployablePair()

	reportDeployerMessage("test", pair, nil, nil, nil, logging.InformationLevel, logger)

	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)
	assert.Equal(t, logCalls[0].PassedArgs().Get(0), logging.InformationLevel)

	logMessage := logCalls[0].PassedArgs().Get(1).(deployerMessage)
	//spew.Dump(logCalls[0].PassedArgs().Get(0))
	//spew.Dump(logCalls[0].PassedArgs().Get(1))

	expectedFields := map[string]interface{}{
		"@loglov3-otl": "sous-rectifier-singularity-v1",
	}
	//replaces logging.StandardVariableFields, look at jims changes in master around this
	variableFields := append(logging.StandardVariableFields,"sous-manifest-id",
		"sous-deployment-diffs",
			"sous-deployment-id",
				"sous-diff-disposition",
					"sous-post-artifact-name",
						"sous-post-artifact-qualities",
							"sous-post-artifact-type",)

	logging.AssertMessageFields(t, logMessage, variableFields, expectedFields)
	//fmt.Printf("msg2: %v\n", logCalls[0].PassedArgs().Get(2))

	//weak check on WriteToConsole
	consoleCalls := control.CallsTo("Console")
	require.Len(t, consoleCalls, 1)
}
