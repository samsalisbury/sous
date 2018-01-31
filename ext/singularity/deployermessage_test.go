package singularity

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
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
	spew.Dump(logCalls[0].PassedArgs().Get(0))
	spew.Dump(logCalls[0].PassedArgs().Get(1))

	logging.AssertMessageFields(t, logMessage, logging.StandardVariableFields, map[string]interface{}{"msg": "test"})
	//fmt.Printf("msg2: %v\n", logCalls[0].PassedArgs().Get(2))

	//weak check on WriteToConsole
	consoleCalls := control.CallsTo("Console")
	require.Len(t, consoleCalls, 1)
}
