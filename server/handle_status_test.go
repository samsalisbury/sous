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

	logCalls := control.CallsTo("Fields")
	require.Len(t, logCalls, 1)
	msgs := logCalls[0].PassedArgs().Get(0).([]logging.EachFielder)

	fixedFields := map[string]interface{}{
		"@loglov3-otl": logging.SousGenericV1,
		"severity":     logging.DebugLevel,
	}

	// treating call-stack-function as variable because the changes to function names could affect it
	logging.AssertMessageFieldlist(t, msgs, append(logging.StandardVariableFields, "call-stack-function", "call-stack-message"), fixedFields)

	assert.Equal(status, 200)
	assert.Len(data.(statusData).Deployments, 0)
}
