package singularity

import (
	"testing"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestDeployerMessage(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()
	pair := baseDeployablePair()

	logDeployerMessage("test", logger, pair)

	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)
	assert.Equal(t, logCalls[0].PassedArgs().Get(0), logging.InformationLevel)
}

