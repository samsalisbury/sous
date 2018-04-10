package sous

import (
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func TestDeliversDiffMessages(t *testing.T) {
	spy, ctrl := logging.NewLogSinkSpy()
	proc := loggingProcessor{ls: spy}
	proc.doLog(&DeployablePair{})

	assert.Len(t, ctrl.CallsTo("Fields"), 1)
	// message for errors

	proc.HandleResolution(&DiffResolution{})
	assert.Len(t, ctrl.CallsTo("Fields"), 2)
}
