package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type terribadLogMessage struct{}

func (msg terribadLogMessage) Message() string {
	panic("never panic while logging; it's not worth crashing the app!")
}
func (msg terribadLogMessage) DefaultLevel() Level {
	panic("never panic while logging; it's not worth crashing the app!")
}
func (msg terribadLogMessage) EachField(fn FieldReportFn) {
	panic("never panic while logging; it's not worth crashing the app!")
}

func TestLogMessagePanicking(t *testing.T) {
	log, ctrl := NewLogSinkSpy(true)

	assert.NotPanics(t, func() {
		NewDeliver(log, terribadLogMessage{})
	})

	calls := ctrl.CallsTo("Fields")
	if assert.Len(t, calls, 2) {
		fields := calls[1].PassedArgs().Get(0).([]EachFielder)
		assert.IsType(t, &silentMessageError{}, fields[2])
	}
}
