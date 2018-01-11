package storage

import (
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func TestLogOnlyStateManager(t *testing.T) {
	log, ctrl := logging.NewLogSinkSpy()
	sm := NewLogOnlyStateManager(log)

	_, err := sm.ReadState()
	assert.NoError(t, err)

	err = sm.WriteState(sous.NewState(), sous.User{})
	assert.NoError(t, err)

	assert.Len(t, ctrl.Calls(), 2)
}
