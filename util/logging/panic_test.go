package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type terribadLogMessage struct{}

func (msg terribadLogMessage) DefaultLevel() Level {
	panic("terribad!")
}

func (msg terribadLogMessage) Message() string {
	panic("so much terrible")
}

func (msg terribadLogMessage) EachField(fn FieldReportFn) {
	panic("never panic while logging; it's not worth crashing the app!")
}

func TestLogMessagePanicking(t *testing.T) {
	log, ctrl := NewLogSinkSpy()

	assert.NotPanics(t, func() {
		Deliver(terribadLogMessage{}, log)
	})

	calls := ctrl.CallsTo("LogMessage")
	if assert.Len(t, calls, 1) {
		assert.IsType(t, &silentMessageError{}, calls[0].PassedArgs().Get(1))
	}
}
