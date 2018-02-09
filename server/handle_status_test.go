package server

import (
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlesStatusGet(t *testing.T) {
	assert := assert.New(t)

	logger, control := logging.NewLogSinkSpy()
	th := &StatusHandler{
		AutoResolver: &sous.AutoResolver{
			GDM:     sous.NewDeployments(),
			LogSink: logger,
		},
	}
	data, status := th.Exchange()

	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)

	logLvl := logCalls[0].PassedArgs().Get(0).(logging.Level)
	msg := logCalls[0].PassedArgs().Get(1).(logging.LogMessage)

	assert.Equal(logLvl, logging.DebugLevel)
	assert.Contains(msg.Message(), "Reporting statuses")

	fixedFields := map[string]interface{}{
		"@loglov3-otl": "sous-generic-v1",
	}

	// treating call-stack-function as variable because the changes to function names could affect it
	logging.AssertMessageFields(t, msg, append(logging.StandardVariableFields, "call-stack-function"), fixedFields)

	assert.Equal(status, 200)
	assert.Len(data.(statusData).Deployments, 0)
}
