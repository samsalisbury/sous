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
	log, _ := NewLogSinkSpy()

	assert.Panics(t, func() {
		Deliver(terribadLogMessage{}, log)
	})

}
